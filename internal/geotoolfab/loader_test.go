package geotoolfab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoad_ResolvesWorkspaceGeoToolDefinitions(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "sinuosity.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifestPath := filepath.Join(workspace, "tools", "geo", "geo_sinuosity.yaml")
	manifest := `name: geo_sinuosity
description: Compute sinuosity ratio for river centerlines.
runner: python
script: tools/geo/scripts/sinuosity.py
timeout_seconds: 45
parameters:
  input_path:
    type: string
    description: Path to the input vector file.
    required: true
  tolerance:
    type: number
    description: Simplify tolerance before measurement.
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loader := NewLoader(workspace)
	defs, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}

	def := defs[0]
	if def.Name != "geo_sinuosity" {
		t.Fatalf("unexpected tool name %q", def.Name)
	}
	if def.TimeoutSeconds != 45 {
		t.Fatalf("expected timeout 45, got %d", def.TimeoutSeconds)
	}
	if def.ScriptPath != scriptPath {
		t.Fatalf("expected script path %q, got %q", scriptPath, def.ScriptPath)
	}
	if def.Parameters["input_path"].Type != "string" || !def.Parameters["input_path"].Required {
		t.Fatalf("unexpected input_path parameter: %+v", def.Parameters["input_path"])
	}
	if def.Parameters["tolerance"].Type != "number" {
		t.Fatalf("unexpected tolerance parameter: %+v", def.Parameters["tolerance"])
	}
}

func TestLoaderLoad_RejectsInvalidToolName(t *testing.T) {
	workspace := t.TempDir()
	manifestDir := filepath.Join(workspace, "tools", "geo")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	manifestPath := filepath.Join(manifestDir, "bad.yaml")
	manifest := `name: sinuosity
description: Invalid because it does not use the geo_ prefix.
runner: python
script: tools/geo/scripts/sinuosity.py
parameters:
  input_path:
    type: string
    required: true
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loader := NewLoader(workspace)
	if _, err := loader.Load(); err == nil {
		t.Fatal("expected invalid tool name to fail")
	}
}

func TestLoaderLoad_RejectsScriptOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	manifestDir := filepath.Join(workspace, "tools", "geo")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	manifestPath := filepath.Join(manifestDir, "geo_escape.yaml")
	manifest := `name: geo_escape
description: Invalid because script escapes the workspace.
runner: python
script: ../outside.py
parameters:
  input_path:
    type: string
    required: true
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loader := NewLoader(workspace)
	if _, err := loader.Load(); err == nil {
		t.Fatal("expected script outside workspace to fail")
	}
}
