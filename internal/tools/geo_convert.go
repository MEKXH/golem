package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GeoFormatConvertInput defines the input for the geo_format_convert tool.
type GeoFormatConvertInput struct {
	InputPath    string `json:"input_path" jsonschema:"required,description=Source geospatial file path"`
	OutputPath   string `json:"output_path" jsonschema:"required,description=Destination file path"`
	OutputFormat string `json:"output_format" jsonschema:"description=GDAL driver name for output format (auto-detected from extension if empty)"`
}

// GeoFormatConvertOutput defines the output of the geo_format_convert tool.
type GeoFormatConvertOutput struct {
	InputPath    string `json:"input_path"`
	OutputPath   string `json:"output_path"`
	InputFormat  string `json:"input_format"`
	OutputFormat string `json:"output_format"`
	Success      bool   `json:"success"`
	Message      string `json:"message"`
}

type geoFormatConvertToolImpl struct {
	gdalBinDir          string
	workspaceDir        string
	timeoutSec          int
	restrictToWorkspace bool
}

func (t *geoFormatConvertToolImpl) execute(ctx context.Context, input *GeoFormatConvertInput) (*GeoFormatConvertOutput, error) {
	inputPath := strings.TrimSpace(input.InputPath)
	outputPath := strings.TrimSpace(input.OutputPath)
	if inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}
	if outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	// Validate paths
	if err := validateGeoFilePath(inputPath, t.workspaceDir, t.restrictToWorkspace); err != nil {
		return nil, fmt.Errorf("input path: %w", err)
	}
	if err := validateGeoFilePath(outputPath, t.workspaceDir, t.restrictToWorkspace); err != nil {
		return nil, fmt.Errorf("output path: %w", err)
	}

	// Detect input format
	inputFmt, inputKnown := detectGeoFormat(inputPath)
	inputFmtName := ""
	if inputKnown {
		inputFmtName = inputFmt.Name
	}

	// Determine output format
	outputFmtStr := strings.TrimSpace(input.OutputFormat)
	outputFmt, outputKnown := detectGeoFormat(outputPath)
	if outputFmtStr == "" && outputKnown {
		outputFmtStr = outputFmt.Name
	}
	if outputFmtStr == "" {
		return nil, fmt.Errorf("cannot auto-detect output format from %q; please specify output_format", outputPath)
	}

	// Decide whether to use ogr2ogr (vector) or gdal_translate (raster)
	isVector := false
	if inputKnown && inputFmt.IsVector && !inputFmt.IsRaster {
		isVector = true
	} else if outputKnown && outputFmt.IsVector && !outputFmt.IsRaster {
		isVector = true
	}
	// GPKG can be both; default to vector for GPKG-to-GPKG
	if inputKnown && outputKnown && inputFmt.IsVector && outputFmt.IsVector {
		isVector = true
	}

	var command string
	var args []string
	if isVector {
		command = "ogr2ogr"
		args = []string{"-f", outputFmtStr, outputPath, inputPath}
	} else {
		command = "gdal_translate"
		args = []string{"-of", outputFmtStr, inputPath, outputPath}
	}

	if !isGdalAvailable(t.gdalBinDir, command) {
		return nil, fmt.Errorf("%s", gdalNotFoundError(command))
	}

	stdout, stderr, exitCode, err := runGdalCommand(ctx, t.gdalBinDir, command, args, "", t.timeoutSec)
	if err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", command, err)
	}

	result := &GeoFormatConvertOutput{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		InputFormat:  inputFmtName,
		OutputFormat: outputFmtStr,
	}

	if exitCode != 0 {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = strings.TrimSpace(stdout)
		}
		result.Success = false
		result.Message = fmt.Sprintf("%s failed (exit %d): %s", command, exitCode, msg)
	} else {
		result.Success = true
		result.Message = fmt.Sprintf("Successfully converted %s → %s (format: %s)", inputPath, outputPath, outputFmtStr)
	}

	return result, nil
}

// NewGeoFormatConvertTool creates the geo_format_convert tool for converting between geospatial formats.
func NewGeoFormatConvertTool(gdalBinDir, workspaceDir string, timeoutSec int, restrictToWorkspace bool) (tool.InvokableTool, error) {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	impl := &geoFormatConvertToolImpl{
		gdalBinDir:          gdalBinDir,
		workspaceDir:        workspaceDir,
		timeoutSec:          timeoutSec,
		restrictToWorkspace: restrictToWorkspace,
	}
	return utils.InferTool(
		"geo_format_convert",
		"Convert a geospatial file between formats (e.g. Shapefile → GeoJSON, GeoTIFF → PNG). "+
			"Auto-detects raster vs vector and uses gdal_translate or ogr2ogr accordingly.",
		impl.execute,
	)
}
