package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func TestGeoCrsDetectTool_PathValidation(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewGeoCrsDetectTool("", workspace, true)
	if err != nil {
		t.Fatalf("NewGeoCrsDetectTool error: %v", err)
	}

	outsidePath := filepath.Join(filepath.Dir(workspace), "outside", "test.tif")
	argsJSON := fmt.Sprintf(`{"path": %q}`, outsidePath)

	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Error("expected error for path outside workspace, got nil")
	}
}

func TestGeoCrsDetectTool_EmptyPath(t *testing.T) {
	tool, err := NewGeoCrsDetectTool("", t.TempDir(), false)
	if err != nil {
		t.Fatalf("NewGeoCrsDetectTool error: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"path": ""}`)
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestGeoCrsDetectTool_ToolInfo(t *testing.T) {
	tool, err := NewGeoCrsDetectTool("", t.TempDir(), false)
	if err != nil {
		t.Fatalf("NewGeoCrsDetectTool error: %v", err)
	}

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if info.Name != "geo_crs_detect" {
		t.Errorf("expected tool name 'geo_crs_detect', got %q", info.Name)
	}
}

func TestParseCrsOutput(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantCRS      string
		wantProjName string
		wantGeog     bool
		wantUnit     string
	}{
		{
			name: "WGS84 geographic",
			raw: `GEOGCRS["WGS 84",
				DATUM["World Geodetic System 1984"],
				ANGLEUNIT["degree",0.0174532925199433],
				ID["EPSG",4326]]`,
			wantCRS:      "EPSG:4326",
			wantProjName: "WGS 84",
			wantGeog:     true,
			wantUnit:     "degree",
		},
		{
			name: "UTM projected",
			raw: `PROJCRS["WGS 84 / UTM zone 50N",
				BASEGEOGCRS["WGS 84"],
				LENGTHUNIT["metre",1],
				ID["EPSG",32650]]`,
			wantCRS:      "EPSG:32650",
			wantProjName: "WGS 84 / UTM zone 50N",
			wantGeog:     false,
			wantUnit:     "metre",
		},
		{
			name: "CGCS2000 old style",
			raw: `GEOGCS["China Geodetic Coordinate System 2000",
				AUTHORITY["EPSG","4490"],
				UNIT["degree",0.0174532925199433]]`,
			wantCRS:      "EPSG:4490",
			wantProjName: "China Geodetic Coordinate System 2000",
			wantGeog:     true,
			wantUnit:     "degree",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := &GeoCrsDetectOutput{}
			parseCrsOutput(output, tc.raw)

			if output.CRS != tc.wantCRS {
				t.Errorf("CRS = %q, want %q", output.CRS, tc.wantCRS)
			}
			if output.ProjName != tc.wantProjName {
				t.Errorf("ProjName = %q, want %q", output.ProjName, tc.wantProjName)
			}
			if output.IsGeographic != tc.wantGeog {
				t.Errorf("IsGeographic = %v, want %v", output.IsGeographic, tc.wantGeog)
			}
			if output.Unit != tc.wantUnit {
				t.Errorf("Unit = %q, want %q", output.Unit, tc.wantUnit)
			}
		})
	}
}
