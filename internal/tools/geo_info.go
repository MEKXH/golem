package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GeoInfoInput defines the input for the geo_info tool.
type GeoInfoInput struct {
	Path string `json:"path" jsonschema:"required,description=Path to the geospatial file to inspect (raster or vector)"`
}

// GeoInfoOutput defines the output of the geo_info tool.
type GeoInfoOutput struct {
	Path      string `json:"path"`
	Format    string `json:"format"`
	CRS       string `json:"crs"`
	Extent    string `json:"extent"`
	Size      string `json:"size"`
	RawOutput string `json:"raw_output"`
}

var (
	gdalInfoDriverRe = regexp.MustCompile(`(?i)Driver:\s*(.+)`)
	gdalInfoCRSRe    = regexp.MustCompile(`(?i)(?:AUTHORITY\["EPSG","(\d+)"\]|ID\["EPSG",(\d+)\])`)
	gdalInfoSizeRe   = regexp.MustCompile(`(?i)Size is (\d+)\s*,\s*(\d+)`)
	gdalInfoExtentRe = regexp.MustCompile(`(?i)(?:Upper Left|Lower Left|Upper Right|Lower Right)\s*\(\s*([^)]+)\)`)
	ogrInfoFeatRe    = regexp.MustCompile(`(?i)Feature Count:\s*(\d+)`)
)

type geoInfoToolImpl struct {
	gdalBinDir          string
	workspaceDir        string
	restrictToWorkspace bool
}

func (t *geoInfoToolImpl) execute(ctx context.Context, input *GeoInfoInput) (*GeoInfoOutput, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if err := validateGeoFilePath(path, t.workspaceDir, t.restrictToWorkspace); err != nil {
		return nil, err
	}

	format, isKnown := detectGeoFormat(path)
	useOgr := isKnown && format.IsVector && !format.IsRaster

	var command string
	var args []string
	if useOgr {
		command = "ogrinfo"
		args = []string{"-al", "-so", path}
	} else {
		command = "gdalinfo"
		args = []string{path}
	}

	if !isGdalAvailable(t.gdalBinDir, command) {
		return nil, fmt.Errorf("%s", gdalNotFoundError(command))
	}

	stdout, stderr, exitCode, err := runGdalCommand(ctx, t.gdalBinDir, command, args, "", 60)
	if err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", command, err)
	}
	if exitCode != 0 {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = strings.TrimSpace(stdout)
		}
		return nil, fmt.Errorf("%s exited with code %d: %s", command, exitCode, msg)
	}

	output := &GeoInfoOutput{
		Path:      path,
		RawOutput: strings.TrimSpace(stdout),
	}

	parseGeoInfoOutput(output, stdout, useOgr)
	if isKnown {
		output.Format = format.Name
	}

	return output, nil
}

func parseGeoInfoOutput(output *GeoInfoOutput, raw string, isVector bool) {
	if m := gdalInfoDriverRe.FindStringSubmatch(raw); len(m) > 1 {
		if output.Format == "" {
			output.Format = strings.TrimSpace(m[1])
		}
	}

	if m := gdalInfoCRSRe.FindStringSubmatch(raw); len(m) > 1 {
		epsg := m[1]
		if epsg == "" {
			epsg = m[2]
		}
		if epsg != "" {
			output.CRS = "EPSG:" + epsg
		}
	}
	if output.CRS == "" && strings.Contains(raw, "WGS 84") {
		output.CRS = "EPSG:4326"
	}

	if isVector {
		if m := ogrInfoFeatRe.FindStringSubmatch(raw); len(m) > 1 {
			output.Size = m[1] + " features"
		}
	} else {
		if m := gdalInfoSizeRe.FindStringSubmatch(raw); len(m) > 2 {
			output.Size = m[1] + " x " + m[2] + " pixels"
		}
	}

	corners := gdalInfoExtentRe.FindAllStringSubmatch(raw, 4)
	if len(corners) >= 2 {
		var parts []string
		for _, c := range corners {
			if len(c) > 1 {
				parts = append(parts, strings.TrimSpace(c[1]))
			}
		}
		output.Extent = strings.Join(parts, " | ")
	}
}

// NewGeoInfoTool creates the geo_info tool for inspecting geospatial files.
func NewGeoInfoTool(gdalBinDir, workspaceDir string, restrictToWorkspace bool) (tool.InvokableTool, error) {
	impl := &geoInfoToolImpl{
		gdalBinDir:          gdalBinDir,
		workspaceDir:        workspaceDir,
		restrictToWorkspace: restrictToWorkspace,
	}
	return utils.InferTool(
		"geo_info",
		"Inspect a geospatial file (raster or vector) and return metadata: format, CRS, extent, and size. Uses gdalinfo/ogrinfo.",
		impl.execute,
	)
}
