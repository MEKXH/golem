package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeoProcessTool_WhitelistEnforced(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewGeoProcessTool("", workspace, 60, true)
	if err != nil {
		t.Fatalf("NewGeoProcessTool error: %v", err)
	}

	// Attempt to use a non-whitelisted command
	argsJSON := `{"command": "rm", "args": ["-rf", "/"]}`
	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Error("expected error for non-whitelisted command, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestGeoProcessTool_ShellInjectionBlocked(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewGeoProcessTool("", workspace, 60, true)
	if err != nil {
		t.Fatalf("NewGeoProcessTool error: %v", err)
	}

	injectionCases := []struct {
		name string
		args []string
	}{
		{"semicolon", []string{"-of", "GTiff; rm -rf /", "in.tif", "out.tif"}},
		{"pipe", []string{"in.tif", "| cat /etc/passwd"}},
		{"backtick", []string{"`whoami`", "out.tif"}},
		{"dollar paren", []string{"$(id)", "out.tif"}},
	}

	for _, tc := range injectionCases {
		t.Run(tc.name, func(t *testing.T) {
			argsBytes, _ := json.Marshal(tc.args)
			argsJSON := fmt.Sprintf(`{"command": "gdal_translate", "args": %s}`, string(argsBytes))

			result, err := tool.InvokableRun(context.Background(), argsJSON)
			if err != nil {
				t.Fatalf("InvokableRun error: %v", err)
			}

			var out GeoProcessOutput
			if err := json.Unmarshal([]byte(result), &out); err != nil {
				t.Fatalf("failed to unmarshal: %v, raw: %s", err, result)
			}

			if out.ExitCode == 0 {
				t.Error("expected non-zero exit code for shell injection attempt")
			}
			if !strings.Contains(out.Stderr, "Blocked") {
				t.Errorf("expected stderr to contain 'Blocked', got: %s", out.Stderr)
			}
		})
	}
}

func TestGeoProcessTool_OutputPathRestriction(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewGeoProcessTool("", workspace, 60, true)
	if err != nil {
		t.Fatalf("NewGeoProcessTool error: %v", err)
	}

	outsidePath := filepath.Join(filepath.Dir(workspace), "evil.tif")
	inputPath := filepath.Join(workspace, "in.tif")
	argsJSON := fmt.Sprintf(`{"command": "gdal_translate", "args": ["-of", "GTiff", %q, %q]}`, inputPath, outsidePath)

	result, err := tool.InvokableRun(context.Background(), argsJSON)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}

	var out GeoProcessOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("failed to unmarshal: %v, raw: %s", err, result)
	}

	if out.ExitCode == 0 {
		t.Error("expected non-zero exit code for output path outside workspace")
	}
	if !strings.Contains(out.Stderr, "Path restriction") && !strings.Contains(out.Stderr, "outside workspace") {
		t.Errorf("expected path restriction error, got: %s", out.Stderr)
	}
}

func TestGeoProcessTool_EmptyCommand(t *testing.T) {
	tool, err := NewGeoProcessTool("", t.TempDir(), 60, false)
	if err != nil {
		t.Fatalf("NewGeoProcessTool error: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"command": "", "args": ["test"]}`)
	if err == nil {
		t.Error("expected error for empty command, got nil")
	}
}

func TestGeoProcessTool_EmptyArgs(t *testing.T) {
	tool, err := NewGeoProcessTool("", t.TempDir(), 60, false)
	if err != nil {
		t.Fatalf("NewGeoProcessTool error: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"command": "gdalinfo", "args": []}`)
	if err == nil {
		t.Error("expected error for empty args, got nil")
	}
}

func TestGeoProcessTool_ToolInfo(t *testing.T) {
	tool, err := NewGeoProcessTool("", t.TempDir(), 60, false)
	if err != nil {
		t.Fatalf("NewGeoProcessTool error: %v", err)
	}

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if info.Name != "geo_process" {
		t.Errorf("expected tool name 'geo_process', got %q", info.Name)
	}
}
