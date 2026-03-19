package tools

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateGeoFilePath_InWorkspace(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "workspace")

	// Path inside workspace should pass
	path := filepath.Join(workspace, "data", "test.tif")
	if err := validateGeoFilePath(path, workspace, true); err != nil {
		t.Errorf("expected path inside workspace to be valid, got: %v", err)
	}
}

func TestValidateGeoFilePath_OutsideWorkspace(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "workspace")

	outsidePath := filepath.Join(filepath.Dir(workspace), "outside", "test.tif")
	if err := validateGeoFilePath(outsidePath, workspace, true); err == nil {
		t.Error("expected error for path outside workspace, got nil")
	}
}

func TestValidateGeoFilePath_TraversalAttempt(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "workspace")

	traversalPath := filepath.Join(workspace, "..", "secret.tif")
	if err := validateGeoFilePath(traversalPath, workspace, true); err == nil {
		t.Error("expected error for path traversal attempt, got nil")
	}
}

func TestValidateGeoFilePath_NotRestricted(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "workspace")
	outsidePath := "/some/other/path/test.tif"
	if runtime.GOOS == "windows" {
		outsidePath = `C:\some\other\path\test.tif`
	}

	if err := validateGeoFilePath(outsidePath, workspace, false); err != nil {
		t.Errorf("expected unrestricted path to be valid, got: %v", err)
	}
}

func TestValidateGeoFilePath_EmptyPath(t *testing.T) {
	workspace := t.TempDir()
	if err := validateGeoFilePath("", workspace, true); err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestDetectGeoFormat(t *testing.T) {
	tests := []struct {
		path     string
		wantName string
		wantOk   bool
	}{
		{"test.tif", "GTiff", true},
		{"test.TIFF", "GTiff", true},
		{"data.shp", "ESRI Shapefile", true},
		{"data.geojson", "GeoJSON", true},
		{"data.gpkg", "GPKG", true},
		{"data.csv", "CSV", true},
		{"data.kml", "KML", true},
		{"data.parquet", "Parquet", true},
		{"unknown.xyz", "", false},
		{"no_extension", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			info, ok := detectGeoFormat(tc.path)
			if ok != tc.wantOk {
				t.Errorf("detectGeoFormat(%q): ok = %v, want %v", tc.path, ok, tc.wantOk)
			}
			if ok && info.Name != tc.wantName {
				t.Errorf("detectGeoFormat(%q): name = %q, want %q", tc.path, info.Name, tc.wantName)
			}
		})
	}
}

func TestContainsShellMeta(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantMeta bool
	}{
		{"clean args", []string{"-of", "GTiff", "in.tif", "out.tif"}, false},
		{"semicolon", []string{"-of", "GTiff; rm -rf /"}, true},
		{"pipe", []string{"in.tif", "| cat /etc/passwd"}, true},
		{"ampersand", []string{"in.tif", "&& echo pwned"}, true},
		{"backtick", []string{"in.tif", "`whoami`"}, true},
		{"dollar paren", []string{"in.tif", "$(whoami)"}, true},
		{"newline", []string{"in.tif", "out.tif\nrm -rf /"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hasMeta, _ := containsShellMeta(tc.args)
			if hasMeta != tc.wantMeta {
				t.Errorf("containsShellMeta(%v) = %v, want %v", tc.args, hasMeta, tc.wantMeta)
			}
		})
	}
}

func TestResolveGdalCommand(t *testing.T) {
	tests := []struct {
		binDir  string
		command string
	}{
		{"", "gdalinfo"},
		{"/opt/gdal/bin", "gdalinfo"},
	}

	for _, tc := range tests {
		result := resolveGdalCommand(tc.binDir, tc.command)
		if result == "" {
			t.Errorf("resolveGdalCommand(%q, %q) returned empty string", tc.binDir, tc.command)
		}
		if tc.binDir != "" {
			expected := filepath.Join(tc.binDir, tc.command)
			if runtime.GOOS == "windows" {
				expected += ".exe"
			}
			if result != expected {
				t.Errorf("resolveGdalCommand(%q, %q) = %q, want %q", tc.binDir, tc.command, result, expected)
			}
		}
	}
}
