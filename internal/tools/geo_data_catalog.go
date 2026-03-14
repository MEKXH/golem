package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	defaultOverpassEndpoint = "https://overpass-api.de/api/interpreter"
	defaultSTACEndpoint     = "https://earth-search.aws.element84.com/v1/search"
)

// GeoDataCatalogInput defines the input for the geo_data_catalog tool.
type GeoDataCatalogInput struct {
	Action      string            `json:"action" jsonschema:"required,description=Catalog action: local_scan, overpass_search, or stac_search"`
	Path        string            `json:"path" jsonschema:"description=Local path to scan for action=local_scan"`
	BBox        []float64         `json:"bbox" jsonschema:"description=Bounding box as [minLon,minLat,maxLon,maxLat] for remote searches"`
	Tags        map[string]string `json:"tags" jsonschema:"description=OSM tag filters for action=overpass_search"`
	Collections []string          `json:"collections" jsonschema:"description=STAC collections for action=stac_search"`
	Limit       int               `json:"limit" jsonschema:"description=Maximum number of results to return"`
}

// GeoDataCatalogItem describes one catalog result.
type GeoDataCatalogItem struct {
	Source      string    `json:"source"`
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Path        string    `json:"path,omitempty"`
	URL         string    `json:"url,omitempty"`
	Format      string    `json:"format,omitempty"`
	DatasetType string    `json:"dataset_type,omitempty"`
	Collection  string    `json:"collection,omitempty"`
	Geometry    string    `json:"geometry,omitempty"`
	BBox        []float64 `json:"bbox,omitempty"`
}

// GeoDataCatalogOutput is the geo_data_catalog result.
type GeoDataCatalogOutput struct {
	Action string               `json:"action"`
	Query  string               `json:"query,omitempty"`
	Items  []GeoDataCatalogItem `json:"items"`
}

type geoDataCatalogToolImpl struct {
	workspaceDir        string
	restrictToWorkspace bool
	timeoutSec          int
	httpClient          *http.Client
	overpassEndpoint    string
	stacEndpoint        string
}

func (t *geoDataCatalogToolImpl) execute(ctx context.Context, input *GeoDataCatalogInput) (*GeoDataCatalogOutput, error) {
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}
	switch action {
	case "local_scan":
		return t.localScan(input.Path)
	case "overpass_search":
		return t.overpassSearch(ctx, input)
	case "stac_search":
		return t.stacSearch(ctx, input)
	default:
		return nil, fmt.Errorf("unsupported action %q", input.Action)
	}
}

func (t *geoDataCatalogToolImpl) localScan(path string) (*GeoDataCatalogOutput, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		target = t.workspaceDir
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(t.workspaceDir, target)
	}
	if err := validateGeoFilePath(target, t.workspaceDir, t.restrictToWorkspace); err != nil {
		return nil, err
	}

	items := make([]GeoDataCatalogItem, 0)
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, ok := detectGeoFormat(path)
		if !ok {
			return nil
		}
		relPath := path
		if rel, relErr := filepath.Rel(t.workspaceDir, path); relErr == nil {
			relPath = rel
		}
		items = append(items, GeoDataCatalogItem{
			Source:      "local",
			Name:        filepath.Base(path),
			Path:        filepath.ToSlash(relPath),
			Format:      info.Name,
			DatasetType: geoDatasetType(info),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return &GeoDataCatalogOutput{Action: "local_scan", Items: items}, nil
}

func (t *geoDataCatalogToolImpl) overpassSearch(ctx context.Context, input *GeoDataCatalogInput) (*GeoDataCatalogOutput, error) {
	bbox, err := normalizeBBox(input.BBox)
	if err != nil {
		return nil, err
	}
	if len(input.Tags) == 0 {
		return nil, fmt.Errorf("tags are required for action=overpass_search")
	}
	limit := resolveGeoCatalogLimit(input.Limit)
	query := buildOverpassQuery(bbox, input.Tags, limit)
	endpoint := strings.TrimSpace(t.overpassEndpoint)
	if endpoint == "" {
		endpoint = defaultOverpassEndpoint
	}
	body := bytes.NewBufferString("data=" + query)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("overpass request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var decoded struct {
		Elements []struct {
			Type   string             `json:"type"`
			ID     int64              `json:"id"`
			Lat    float64            `json:"lat"`
			Lon    float64            `json:"lon"`
			Center map[string]float64 `json:"center"`
			Tags   map[string]string  `json:"tags"`
		} `json:"elements"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("parse overpass response: %w", err)
	}

	items := make([]GeoDataCatalogItem, 0, len(decoded.Elements))
	for _, el := range decoded.Elements {
		bboxItem := []float64{}
		if el.Lon != 0 || el.Lat != 0 {
			bboxItem = []float64{el.Lon, el.Lat, el.Lon, el.Lat}
		} else if lon, ok := el.Center["lon"]; ok {
			lat := el.Center["lat"]
			bboxItem = []float64{lon, lat, lon, lat}
		}
		name := el.Tags["name"]
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("%s/%d", el.Type, el.ID)
		}
		items = append(items, GeoDataCatalogItem{
			Source:      "overpass",
			ID:          fmt.Sprintf("%s/%d", el.Type, el.ID),
			Name:        name,
			DatasetType: "vector",
			Geometry:    el.Type,
			BBox:        bboxItem,
		})
	}
	return &GeoDataCatalogOutput{Action: "overpass_search", Query: query, Items: items}, nil
}

func (t *geoDataCatalogToolImpl) stacSearch(ctx context.Context, input *GeoDataCatalogInput) (*GeoDataCatalogOutput, error) {
	bbox, err := normalizeBBox(input.BBox)
	if err != nil {
		return nil, err
	}
	if len(input.Collections) == 0 {
		return nil, fmt.Errorf("collections are required for action=stac_search")
	}
	limit := resolveGeoCatalogLimit(input.Limit)
	payload := map[string]any{
		"collections": input.Collections,
		"bbox":        bbox,
		"limit":       limit,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimSpace(t.stacEndpoint)
	if endpoint == "" {
		endpoint = defaultSTACEndpoint
	}
	if !strings.HasSuffix(strings.ToLower(endpoint), "/search") {
		endpoint = strings.TrimRight(endpoint, "/") + "/search"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("stac request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var decoded struct {
		Features []struct {
			ID         string                       `json:"id"`
			Collection string                       `json:"collection"`
			Assets     map[string]map[string]string `json:"assets"`
		} `json:"features"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("parse stac response: %w", err)
	}

	items := make([]GeoDataCatalogItem, 0, len(decoded.Features))
	for _, feature := range decoded.Features {
		assetURL := ""
		format := ""
		for _, asset := range feature.Assets {
			if href := strings.TrimSpace(asset["href"]); href != "" {
				assetURL = href
				if info, ok := detectGeoFormat(href); ok {
					format = info.Name
				}
				break
			}
		}
		items = append(items, GeoDataCatalogItem{
			Source:      "stac",
			ID:          feature.ID,
			Name:        feature.ID,
			URL:         assetURL,
			Format:      format,
			DatasetType: "raster",
			Collection:  feature.Collection,
		})
	}
	return &GeoDataCatalogOutput{Action: "stac_search", Query: string(body), Items: items}, nil
}

func (t *geoDataCatalogToolImpl) client() *http.Client {
	if t.httpClient != nil {
		return t.httpClient
	}
	timeout := time.Duration(t.timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func resolveGeoCatalogLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func normalizeBBox(bbox []float64) ([]float64, error) {
	if len(bbox) != 4 {
		return nil, fmt.Errorf("bbox must contain 4 numbers: [minLon,minLat,maxLon,maxLat]")
	}
	return bbox, nil
}

func buildOverpassQuery(bbox []float64, tags map[string]string, limit int) string {
	parts := make([]string, 0, len(tags))
	for key, value := range tags {
		parts = append(parts, fmt.Sprintf("[\"%s\"=\"%s\"]", key, value))
	}
	sort.Strings(parts)
	tagExpr := strings.Join(parts, "")
	south, west, north, east := bbox[1], bbox[0], bbox[3], bbox[2]
	return fmt.Sprintf("[out:json][timeout:25];(node%s(%f,%f,%f,%f);way%s(%f,%f,%f,%f);relation%s(%f,%f,%f,%f););out center tags %d;", tagExpr, south, west, north, east, tagExpr, south, west, north, east, tagExpr, south, west, north, east, limit)
}

func geoDatasetType(info geoFormatInfo) string {
	if info.IsRaster && info.IsVector {
		return "raster+vector"
	}
	if info.IsRaster {
		return "raster"
	}
	if info.IsVector {
		return "vector"
	}
	return "unknown"
}

// NewGeoDataCatalogTool creates the geo_data_catalog tool for local and remote geospatial dataset discovery.
func NewGeoDataCatalogTool(workspaceDir string, restrictToWorkspace bool, timeoutSec int) (tool.InvokableTool, error) {
	impl := &geoDataCatalogToolImpl{
		workspaceDir:        workspaceDir,
		restrictToWorkspace: restrictToWorkspace,
		timeoutSec:          timeoutSec,
		overpassEndpoint:    defaultOverpassEndpoint,
		stacEndpoint:        defaultSTACEndpoint,
	}
	return utils.InferTool(
		"geo_data_catalog",
		"Discover geospatial datasets from the local workspace, OpenStreetMap Overpass, or a STAC API.",
		impl.execute,
	)
}
