package tools

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/geotoolfab"
)

func TestGeoFabricatedTool_ExecutesRunnerWithJSONStdin(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "helper.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("# helper placeholder\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	def := geotoolfab.Definition{
		Name:        "geo_sinuosity",
		Description: "Compute sinuosity ratio for a river centerline.",
		Runner:      os.Args[0],
		ScriptPath:  scriptPath,
		Args:        []string{"-test.run=TestGeoFabricatedToolHelperProcess", "--", "geo-tool-helper"},
		Parameters: map[string]geotoolfab.Parameter{
			"input_path": {Type: "string", Required: true, Description: "Path to the input line dataset."},
			"tolerance":  {Type: "number", Description: "Simplification tolerance."},
		},
		TimeoutSeconds: 30,
	}

	externalTool, err := NewGeoFabricatedTool(def)
	if err != nil {
		t.Fatalf("NewGeoFabricatedTool() error = %v", err)
	}

	out, err := externalTool.InvokableRun(context.Background(), `{"input_path":"data/river.geojson","tolerance":2.5}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if !strings.Contains(out, `"input_path":"data/river.geojson"`) {
		t.Fatalf("expected tool output to include stdin payload, got %s", out)
	}
	if !strings.Contains(out, `"script":"`+filepath.ToSlash(scriptPath)+`"`) {
		t.Fatalf("expected tool output to include script path, got %s", out)
	}
}

func TestGeoFabricatedTool_RejectsMissingRequiredParameter(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "helper.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("# helper placeholder\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	def := geotoolfab.Definition{
		Name:        "geo_sinuosity",
		Description: "Compute sinuosity ratio for a river centerline.",
		Runner:      "python",
		ScriptPath:  scriptPath,
		Parameters: map[string]geotoolfab.Parameter{
			"input_path": {Type: "string", Required: true, Description: "Path to the input line dataset."},
		},
		TimeoutSeconds: 30,
	}

	externalTool, err := NewGeoFabricatedTool(def)
	if err != nil {
		t.Fatalf("NewGeoFabricatedTool() error = %v", err)
	}

	if _, err := externalTool.InvokableRun(context.Background(), `{}`); err == nil {
		t.Fatal("expected missing required parameter to fail")
	}
}

func TestGeoFabricatedToolHelperProcess(t *testing.T) {
	for _, arg := range os.Args {
		if arg != "geo-tool-helper" {
			continue
		}

		payload, err := io.ReadAll(os.Stdin)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		var input map[string]any
		if err := json.Unmarshal(payload, &input); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		result := map[string]any{
			"input":  input,
			"script": filepath.ToSlash(os.Args[len(os.Args)-1]),
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
		os.Exit(0)
	}
}

func TestPrepareGeoFabricatedInvocation_ValidatesArguments(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "helper.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("# helper placeholder\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := prepareGeoFabricatedInvocation(geotoolfab.Definition{
		Name:        "geo_sinuosity",
		Description: "Compute sinuosity ratio for a river centerline.",
		Runner:      "python",
		ScriptPath:  scriptPath,
		Parameters: map[string]geotoolfab.Parameter{
			"input_path": {Type: "string", Required: true, Description: "Path to the input line dataset."},
		},
		TimeoutSeconds: 30,
	}, `{}`)
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestPrepareGeoFabricatedInvocation_NormalizesEmptyPayload(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "helper.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("# helper placeholder\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	invocation, err := prepareGeoFabricatedInvocation(geotoolfab.Definition{
		Name:           "geo_sinuosity",
		Description:    "Compute sinuosity ratio for a river centerline.",
		Runner:         "python",
		ScriptPath:     scriptPath,
		Args:           []string{"-u"},
		TimeoutSeconds: 15,
	}, "")
	if err != nil {
		t.Fatalf("prepareGeoFabricatedInvocation() error = %v", err)
	}
	if invocation.payload != "{}" {
		t.Fatalf("expected normalized empty payload, got %q", invocation.payload)
	}
	if len(invocation.args) != 2 || invocation.args[0] != "-u" || invocation.args[1] != scriptPath {
		t.Fatalf("unexpected invocation args %+v", invocation.args)
	}
	if invocation.timeout != 15*time.Second {
		t.Fatalf("expected 15 second timeout, got %s", invocation.timeout)
	}
}

func TestBuildGeoFabricationDryRun_PackagesScaffold(t *testing.T) {
	workspace := t.TempDir()
	dryRun, err := BuildGeoFabricationDryRun(workspace, geotoolfab.ScaffoldSpec{
		Name:        "sinuosity",
		Description: "Compute sinuosity ratio for a river centerline.",
		Runner:      "python",
		Parameters: map[string]geotoolfab.Parameter{
			"input_path": {Type: "string", Required: true, Description: "Input line dataset path."},
		},
	})
	if err != nil {
		t.Fatalf("BuildGeoFabricationDryRun() error = %v", err)
	}
	if !dryRun.ValidationPassed {
		t.Fatal("expected dry-run fabrication bundle to report a valid scaffold")
	}
	if filepath.Base(dryRun.ManifestPath) != "geo_sinuosity.yaml" {
		t.Fatalf("unexpected manifest path %q", dryRun.ManifestPath)
	}
	if filepath.Base(dryRun.ScriptPath) != "geo_sinuosity.py" {
		t.Fatalf("unexpected script path %q", dryRun.ScriptPath)
	}
	if !strings.Contains(dryRun.ManifestBody, "name: geo_sinuosity") {
		t.Fatalf("expected manifest body to include normalized tool name, got %s", dryRun.ManifestBody)
	}
	if !strings.Contains(dryRun.ScriptBody, "def main()") {
		t.Fatalf("expected script body to include python stub, got %s", dryRun.ScriptBody)
	}
}
