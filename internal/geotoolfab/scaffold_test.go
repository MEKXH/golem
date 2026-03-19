package geotoolfab

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildScaffold_GeneratesValidatorCompliantTool(t *testing.T) {
	workspace := t.TempDir()
	scaffold, err := BuildScaffold(workspace, ScaffoldSpec{
		Name:        "sinuosity",
		Description: "Compute sinuosity ratio for a river centerline.",
		Runner:      "python",
		Parameters: map[string]Parameter{
			"input_path": {Type: "string", Required: true, Description: "Input line dataset path."},
			"tolerance":  {Type: "number", Description: "Simplification tolerance."},
		},
	})
	if err != nil {
		t.Fatalf("BuildScaffold() error = %v", err)
	}
	if scaffold.ToolName != "geo_sinuosity" {
		t.Fatalf("expected normalized tool name, got %q", scaffold.ToolName)
	}
	if !strings.Contains(scaffold.ManifestBody, "name: geo_sinuosity") {
		t.Fatalf("expected geo-prefixed manifest, got %s", scaffold.ManifestBody)
	}
	if filepath.Base(scaffold.ManifestPath) != "geo_sinuosity.yaml" {
		t.Fatalf("unexpected manifest path %q", scaffold.ManifestPath)
	}
	if filepath.Base(scaffold.ScriptPath) != "geo_sinuosity.py" {
		t.Fatalf("unexpected script path %q", scaffold.ScriptPath)
	}
	if !strings.Contains(scaffold.ScriptBody, "def main()") {
		t.Fatalf("expected python script stub, got %s", scaffold.ScriptBody)
	}
	if !scaffold.ValidationPassed {
		t.Fatal("expected scaffold to report validator success")
	}
}

func TestBuildScaffold_RejectsUnsupportedParameterTypes(t *testing.T) {
	workspace := t.TempDir()
	_, err := BuildScaffold(workspace, ScaffoldSpec{
		Name:        "sinuosity",
		Description: "Compute sinuosity ratio for a river centerline.",
		Parameters: map[string]Parameter{
			"target_table": {Type: "table"},
		},
	})
	if err == nil {
		t.Fatal("expected unsupported parameter type error")
	}
}
