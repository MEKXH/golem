package geotoolfab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDefinition_NormalizesDefaults(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "helper.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	def, err := ValidateDefinition(Definition{
		Name:        "geo_sinuosity",
		Description: "Compute sinuosity ratio.",
		ScriptPath:  scriptPath,
		Parameters: map[string]Parameter{
			"input_path": {Required: true},
			"tolerance":  {Type: " NUMBER "},
		},
	})
	if err != nil {
		t.Fatalf("ValidateDefinition() error = %v", err)
	}
	if def.Runner != "python" {
		t.Fatalf("expected default runner python, got %q", def.Runner)
	}
	if def.TimeoutSeconds != defaultTimeoutSeconds {
		t.Fatalf("expected default timeout %d, got %d", defaultTimeoutSeconds, def.TimeoutSeconds)
	}
	if def.Parameters["input_path"].Type != "string" {
		t.Fatalf("expected default parameter type string, got %+v", def.Parameters["input_path"])
	}
	if def.Parameters["tolerance"].Type != "number" {
		t.Fatalf("expected normalized parameter type number, got %+v", def.Parameters["tolerance"])
	}
}

func TestValidateDefinition_ReturnsErrorForMissingScript(t *testing.T) {
	_, err := ValidateDefinition(Definition{
		Name:        "geo_sinuosity",
		Description: "Compute sinuosity ratio.",
		Runner:      "python",
		ScriptPath:  filepath.Join(t.TempDir(), "missing.py"),
	})
	if err == nil {
		t.Fatal("expected missing script validation error")
	}
}

func TestValidateDefinition_ReturnsErrorForUnsupportedParameterType(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "helper.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := ValidateDefinition(Definition{
		Name:        "geo_sinuosity",
		Description: "Compute sinuosity ratio.",
		Runner:      "python",
		ScriptPath:  scriptPath,
		Parameters: map[string]Parameter{
			"input_path": {Type: "table"},
		},
	})
	if err == nil {
		t.Fatal("expected unsupported parameter type validation error")
	}
}
