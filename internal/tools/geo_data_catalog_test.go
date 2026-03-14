package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGeoDataCatalogTool_LocalScanFindsGeoFiles(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "data", "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	for path, content := range map[string]string{
		filepath.Join(workspace, "data", "roads.geojson"):        "{}",
		filepath.Join(workspace, "data", "dem.tif"):              "raster",
		filepath.Join(workspace, "data", "nested", "ignore.txt"): "text",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}

	tool := &geoDataCatalogToolImpl{workspaceDir: workspace, restrictToWorkspace: true, timeoutSec: 15}
	out, err := tool.execute(context.Background(), &GeoDataCatalogInput{Action: "local_scan", Path: filepath.Join(workspace, "data")})
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 geo files, got %+v", out.Items)
	}
	if out.Items[0].Source != "local" {
		t.Fatalf("expected local source, got %+v", out.Items[0])
	}
	if out.Items[0].Path == "" || out.Items[1].Format == "" {
		t.Fatalf("expected populated local scan items, got %+v", out.Items)
	}
}

func TestGeoDataCatalogTool_LocalScanRestrictsOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	tool := &geoDataCatalogToolImpl{workspaceDir: workspace, restrictToWorkspace: true, timeoutSec: 15}
	_, err := tool.execute(context.Background(), &GeoDataCatalogInput{Action: "local_scan", Path: filepath.Join(filepath.Dir(workspace), "outside")})
	if err == nil {
		t.Fatal("expected workspace restriction error")
	}
}

func TestGeoDataCatalogTool_OverpassSearchParsesElements(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"elements":[{"type":"node","id":1,"lat":39.9,"lon":116.3,"tags":{"name":"School A","amenity":"school"}},{"type":"way","id":2,"center":{"lat":39.91,"lon":116.31},"tags":{"name":"School B","amenity":"school"}}]}`))
	}))
	defer server.Close()

	tool := &geoDataCatalogToolImpl{httpClient: &http.Client{Timeout: 5 * time.Second}, overpassEndpoint: server.URL, stacEndpoint: "http://invalid", timeoutSec: 15}
	out, err := tool.execute(context.Background(), &GeoDataCatalogInput{Action: "overpass_search", BBox: []float64{116.2, 39.8, 116.4, 40.0}, Tags: map[string]string{"amenity": "school"}, Limit: 5})
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 overpass items, got %+v", out.Items)
	}
	if out.Items[0].Source != "overpass" || out.Items[0].Name == "" {
		t.Fatalf("unexpected overpass item %+v", out.Items[0])
	}
	if out.Query == "" {
		t.Fatalf("expected stored overpass query, got %+v", out)
	}
}

func TestGeoDataCatalogTool_STACSearchParsesFeatures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"features":[{"id":"sentinel-1","collection":"sentinel-2-l2a","assets":{"visual":{"href":"https://example.test/visual.tif"}}},{"id":"sentinel-2","collection":"sentinel-2-l2a","assets":{"thumbnail":{"href":"https://example.test/thumb.jpg"}}}]}`))
	}))
	defer server.Close()

	tool := &geoDataCatalogToolImpl{httpClient: &http.Client{Timeout: 5 * time.Second}, overpassEndpoint: "http://invalid", stacEndpoint: server.URL, timeoutSec: 15}
	out, err := tool.execute(context.Background(), &GeoDataCatalogInput{Action: "stac_search", Collections: []string{"sentinel-2-l2a"}, BBox: []float64{116.2, 39.8, 116.4, 40.0}, Limit: 2})
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 stac items, got %+v", out.Items)
	}
	if out.Items[0].Source != "stac" || out.Items[0].Collection == "" {
		t.Fatalf("unexpected stac item %+v", out.Items[0])
	}
}
