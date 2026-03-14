package tools

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	def := geotoolfab.Definition{
		Name:        "geo_sinuosity",
		Description: "Compute sinuosity ratio for a river centerline.",
		Runner:      "python",
		ScriptPath:  filepath.Join(t.TempDir(), "tools", "geo", "scripts", "helper.py"),
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
