package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func TestGeoFormatConvertTool_FormatDetection(t *testing.T) {
	tests := []struct {
		inputPath  string
		outputPath string
		expectCmd  string // "ogr2ogr" for vector, "gdal_translate" for raster
	}{
		{"data.shp", "data.geojson", "ogr2ogr"},
		{"data.geojson", "data.gpkg", "ogr2ogr"},
		{"image.tif", "image.png", "gdal_translate"},
		{"raster.tif", "raster.jpg", "gdal_translate"},
	}

	for _, tc := range tests {
		t.Run(tc.inputPath+"→"+tc.outputPath, func(t *testing.T) {
			inputFmt, inputOk := detectGeoFormat(tc.inputPath)
			_, outputOk := detectGeoFormat(tc.outputPath)

			if !inputOk || !outputOk {
				t.Fatalf("format detection failed for %s or %s", tc.inputPath, tc.outputPath)
			}

			isVector := inputFmt.IsVector && !inputFmt.IsRaster
			if tc.expectCmd == "ogr2ogr" && !isVector {
				t.Errorf("expected vector format for %s, got raster", tc.inputPath)
			}
			if tc.expectCmd == "gdal_translate" && isVector {
				t.Errorf("expected raster format for %s, got vector", tc.inputPath)
			}
		})
	}
}

func TestGeoFormatConvertTool_OutputPathRestriction(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewGeoFormatConvertTool("", workspace, 60, true)
	if err != nil {
		t.Fatalf("NewGeoFormatConvertTool error: %v", err)
	}

	inputPath := filepath.Join(workspace, "in.shp")
	outsidePath := filepath.Join(filepath.Dir(workspace), "evil.geojson")
	argsJSON := fmt.Sprintf(`{"input_path": %q, "output_path": %q}`, inputPath, outsidePath)

	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Error("expected error for output path outside workspace, got nil")
	}
}

func TestGeoFormatConvertTool_EmptyInputPath(t *testing.T) {
	tool, err := NewGeoFormatConvertTool("", t.TempDir(), 60, false)
	if err != nil {
		t.Fatalf("NewGeoFormatConvertTool error: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"input_path": "", "output_path": "out.geojson"}`)
	if err == nil {
		t.Error("expected error for empty input_path, got nil")
	}
}

func TestGeoFormatConvertTool_UnknownOutputFormat(t *testing.T) {
	tool, err := NewGeoFormatConvertTool("", t.TempDir(), 60, false)
	if err != nil {
		t.Fatalf("NewGeoFormatConvertTool error: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"input_path": "in.shp", "output_path": "out.xyz"}`)
	if err == nil {
		t.Error("expected error for unknown output format, got nil")
	}
}

func TestGeoFormatConvertTool_ToolInfo(t *testing.T) {
	tool, err := NewGeoFormatConvertTool("", t.TempDir(), 60, false)
	if err != nil {
		t.Fatalf("NewGeoFormatConvertTool error: %v", err)
	}

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if info.Name != "geo_format_convert" {
		t.Errorf("expected tool name 'geo_format_convert', got %q", info.Name)
	}
}
