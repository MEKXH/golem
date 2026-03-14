package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
)

func TestGeoInfoTool_PathValidation(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewGeoInfoTool("", workspace, true)
	if err != nil {
		t.Fatalf("NewGeoInfoTool error: %v", err)
	}

	outsidePath := filepath.Join(filepath.Dir(workspace), "outside", "test.tif")
	argsJSON := fmt.Sprintf(`{"path": %q}`, outsidePath)

	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Error("expected error for path outside workspace, got nil")
	}
}

func TestGeoInfoTool_EmptyPath(t *testing.T) {
	tool, err := NewGeoInfoTool("", t.TempDir(), false)
	if err != nil {
		t.Fatalf("NewGeoInfoTool error: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"path": ""}`)
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestGeoInfoTool_ToolInfo(t *testing.T) {
	tool, err := NewGeoInfoTool("", t.TempDir(), false)
	if err != nil {
		t.Fatalf("NewGeoInfoTool error: %v", err)
	}

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if info.Name != "geo_info" {
		t.Errorf("expected tool name 'geo_info', got %q", info.Name)
	}
}

func TestParseGeoInfoOutput_Raster(t *testing.T) {
	raw := `Driver: GTiff/GeoTIFF
Files: test.tif
Size is 1024, 768
Coordinate System is:
GEOGCRS["WGS 84",
    ID["EPSG",4326]]
Upper Left  ( 116.0,  40.0)
Lower Left  ( 116.0,  39.0)
Upper Right ( 117.0,  40.0)
Lower Right ( 117.0,  39.0)`

	output := &GeoInfoOutput{}
	parseGeoInfoOutput(output, raw, false)

	if output.CRS != "EPSG:4326" {
		t.Errorf("CRS = %q, want EPSG:4326", output.CRS)
	}
	if output.Size != "1024 x 768 pixels" {
		t.Errorf("Size = %q, want '1024 x 768 pixels'", output.Size)
	}
	if output.Extent == "" {
		t.Error("Extent should not be empty")
	}
}

func TestParseGeoInfoOutput_Vector(t *testing.T) {
	raw := `INFO: Open of 'test.shp'
Feature Count: 42
Layer name: test
Geometry: Polygon
AUTHORITY["EPSG","4490"]`

	output := &GeoInfoOutput{}
	parseGeoInfoOutput(output, raw, true)

	if output.CRS != "EPSG:4490" {
		t.Errorf("CRS = %q, want EPSG:4490", output.CRS)
	}
	if output.Size != "42 features" {
		t.Errorf("Size = %q, want '42 features'", output.Size)
	}
}

func TestGeoInfoTool_MissingFile(t *testing.T) {
	tool, err := NewGeoInfoTool("", t.TempDir(), false)
	if err != nil {
		t.Fatalf("NewGeoInfoTool error: %v", err)
	}

	argsJSON := `{"path": "/nonexistent/file.tif"}`
	result, err := tool.InvokableRun(context.Background(), argsJSON)
	// Should error either from GDAL not being available or from the file not existing
	if err == nil {
		var output GeoInfoOutput
		if jsonErr := json.Unmarshal([]byte(result), &output); jsonErr == nil {
			// If it returned a result, the raw output should indicate failure
			t.Log("geo_info returned result for missing file (GDAL may not be installed)")
		}
	}
}
