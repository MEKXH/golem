package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GeoCrsDetectInput defines the input for the geo_crs_detect tool.
type GeoCrsDetectInput struct {
	Path string `json:"path" jsonschema:"required,description=Path to the geospatial file to detect CRS for"`
}

// GeoCrsDetectOutput defines the output of the geo_crs_detect tool.
type GeoCrsDetectOutput struct {
	Path         string `json:"path"`
	CRS          string `json:"crs"`
	ProjName     string `json:"proj_name"`
	IsGeographic bool   `json:"is_geographic"`
	Unit         string `json:"unit"`
	RawWKT       string `json:"raw_wkt"`
}

var (
	srsInfoEPSGRe   = regexp.MustCompile(`(?i)(?:EPSG:(\d+)|AUTHORITY\["EPSG"\s*,\s*"?(\d+)"?\]|ID\["EPSG"\s*,\s*(\d+)\])`)
	srsInfoProjRe   = regexp.MustCompile(`(?i)PROJCRS\["([^"]+)"|GEOGCRS\["([^"]+)"|PROJCS\["([^"]+)"|GEOGCS\["([^"]+)"`)
	srsInfoUnitRe   = regexp.MustCompile(`(?i)LENGTHUNIT\["([^"]+)"|ANGLEUNIT\["([^"]+)"|UNIT\["([^"]+)"`)
	srsInfoGeogcsRe = regexp.MustCompile(`(?i)GEOGCRS\[|GEOGCS\[`)
	srsInfoProjcsRe = regexp.MustCompile(`(?i)PROJCRS\[|PROJCS\[`)
)

type geoCrsDetectToolImpl struct {
	gdalBinDir          string
	workspaceDir        string
	restrictToWorkspace bool
}

func (t *geoCrsDetectToolImpl) execute(ctx context.Context, input *GeoCrsDetectInput) (*GeoCrsDetectOutput, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if err := validateGeoFilePath(path, t.workspaceDir, t.restrictToWorkspace); err != nil {
		return nil, err
	}

	if !isGdalAvailable(t.gdalBinDir, "gdalsrsinfo") {
		// Fallback: try gdalinfo
		return t.fallbackWithGdalInfo(ctx, path)
	}

	stdout, stderr, exitCode, err := runGdalCommand(ctx, t.gdalBinDir, "gdalsrsinfo", []string{"-o", "wkt", path}, "", 60)
	if err != nil {
		return nil, fmt.Errorf("failed to run gdalsrsinfo: %w", err)
	}
	if exitCode != 0 {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = strings.TrimSpace(stdout)
		}
		if strings.Contains(strings.ToLower(msg), "unable to open") || strings.Contains(strings.ToLower(msg), "not recognised") {
			return nil, fmt.Errorf("cannot open file %q: %s", path, msg)
		}
		return nil, fmt.Errorf("gdalsrsinfo exited with code %d: %s", exitCode, msg)
	}

	output := &GeoCrsDetectOutput{
		Path:   path,
		RawWKT: strings.TrimSpace(stdout),
	}

	parseCrsOutput(output, stdout)

	// If no EPSG found from WKT, try -o epsg
	if output.CRS == "" {
		epsgStdout, _, _, _ := runGdalCommand(ctx, t.gdalBinDir, "gdalsrsinfo", []string{"-o", "epsg", path}, "", 30)
		if m := srsInfoEPSGRe.FindStringSubmatch(epsgStdout); len(m) > 1 {
			output.CRS = "EPSG:" + m[1]
		}
	}

	return output, nil
}

func (t *geoCrsDetectToolImpl) fallbackWithGdalInfo(ctx context.Context, path string) (*GeoCrsDetectOutput, error) {
	if !isGdalAvailable(t.gdalBinDir, "gdalinfo") {
		return nil, fmt.Errorf("%s", gdalNotFoundError("gdalsrsinfo"))
	}

	stdout, stderr, exitCode, err := runGdalCommand(ctx, t.gdalBinDir, "gdalinfo", []string{path}, "", 60)
	if err != nil {
		return nil, fmt.Errorf("failed to run gdalinfo: %w", err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("gdalinfo exited with code %d: %s", exitCode, strings.TrimSpace(stderr))
	}

	output := &GeoCrsDetectOutput{
		Path:   path,
		RawWKT: "",
	}

	parseCrsOutput(output, stdout)
	return output, nil
}

func parseCrsOutput(output *GeoCrsDetectOutput, raw string) {
	if m := srsInfoEPSGRe.FindStringSubmatch(raw); len(m) > 1 {
		for _, g := range m[1:] {
			if g != "" {
				output.CRS = "EPSG:" + g
				break
			}
		}
	}

	if m := srsInfoProjRe.FindStringSubmatch(raw); len(m) > 1 {
		for _, g := range m[1:] {
			if g != "" {
				output.ProjName = g
				break
			}
		}
	}

	output.IsGeographic = srsInfoGeogcsRe.MatchString(raw) && !srsInfoProjcsRe.MatchString(raw)

	if m := srsInfoUnitRe.FindStringSubmatch(raw); len(m) > 1 {
		for _, g := range m[1:] {
			if g != "" {
				output.Unit = g
				break
			}
		}
	}
}

// NewGeoCrsDetectTool creates the geo_crs_detect tool for detecting CRS of geospatial files.
func NewGeoCrsDetectTool(gdalBinDir, workspaceDir string, restrictToWorkspace bool) (tool.InvokableTool, error) {
	impl := &geoCrsDetectToolImpl{
		gdalBinDir:          gdalBinDir,
		workspaceDir:        workspaceDir,
		restrictToWorkspace: restrictToWorkspace,
	}
	return utils.InferTool(
		"geo_crs_detect",
		"Detect the Coordinate Reference System (CRS) of a geospatial file. "+
			"Returns EPSG code, projection name, whether it is geographic or projected, and the unit.",
		impl.execute,
	)
}
