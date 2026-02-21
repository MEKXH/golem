package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/metrics"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestStatusCommand_PrintsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	output := captureOutput(t, func() {
		if err := runStatus(nil, nil); err != nil {
			t.Fatalf("runStatus error: %v", err)
		}
	})

	cleanOutput := stripANSI(output)

	if !strings.Contains(cleanOutput, "Golem Status") {
		t.Fatalf("expected status output, got: %s", cleanOutput)
	}
	// Updated checks for new layout
	if !strings.Contains(cleanOutput, "Config") || !strings.Contains(cleanOutput, "Path:") {
		t.Fatalf("expected config section, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "Mode") {
		t.Fatalf("expected workspace mode line, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "Tools") {
		t.Fatalf("expected tools section, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "Runtime Metrics") {
		t.Fatalf("expected runtime metrics section, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "no runtime data yet") {
		t.Fatalf("expected no runtime data message, got: %s", cleanOutput)
	}
	// Check for enabled/ready status in a way that tolerates formatting
	if !strings.Contains(cleanOutput, "web_search") || !strings.Contains(cleanOutput, "enabled") {
		t.Fatalf("expected web_search readiness line, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "voice_transcription") || !strings.Contains(cleanOutput, "disabled") {
		t.Fatalf("expected voice transcription status line, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "edit_file") || !strings.Contains(cleanOutput, "ready") {
		t.Fatalf("expected edit/append tool readiness lines, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "workflow") || !strings.Contains(cleanOutput, "ready") {
		t.Fatalf("expected workflow readiness line, got: %s", cleanOutput)
	}
}

func TestStatusCommand_InvalidWorkspaceModeReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	raw := `{
  "agents": {
    "defaults": {
      "workspace_mode": "path",
      "workspace": ""
    }
  }
}`

	if err := os.WriteFile(configPath, []byte(raw), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := runStatus(nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestStatusCommand_PrintsRuntimeMetricsSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	workspacePath := filepath.Join(tmpDir, ".golem", "workspace")
	recorder := metrics.NewRuntimeMetrics(workspacePath)
	_, _ = recorder.RecordToolExecution(123*time.Millisecond, "", nil)
	_, _ = recorder.RecordToolExecution(2*time.Second, "", os.ErrDeadlineExceeded)
	_, _ = recorder.RecordChannelSend(false)
	recorder.Close()

	output := captureOutput(t, func() {
		if err := runStatus(nil, nil); err != nil {
			t.Fatalf("runStatus error: %v", err)
		}
	})

	cleanOutput := stripANSI(output)

	if !strings.Contains(cleanOutput, "tool_total=2") {
		t.Fatalf("expected tool_total in runtime metrics output, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "tool_timeout_ratio=0.500") {
		t.Fatalf("expected timeout ratio in runtime metrics output, got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, "channel_send_failure_ratio=1.000") {
		t.Fatalf("expected channel failure ratio in runtime metrics output, got: %s", cleanOutput)
	}
}

func TestStatusCommand_JSONOutputIncludesRuntimeMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	workspacePath := filepath.Join(tmpDir, ".golem", "workspace")
	recorder := metrics.NewRuntimeMetrics(workspacePath)
	_, _ = recorder.RecordToolExecution(80*time.Millisecond, "", nil)
	_, _ = recorder.RecordChannelSend(true)
	_, _ = recorder.RecordMemoryRecall(2, map[string]int{
		"diary_recent": 2,
	})
	recorder.Close()

	cmd := NewStatusCmd()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set --json: %v", err)
	}

	output := captureOutput(t, func() {
		if err := runStatus(cmd, nil); err != nil {
			t.Fatalf("runStatus error: %v", err)
		}
	})

	// JSON output should not have ANSI codes, but we can check the parsed object
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("invalid json output: %v, output=%s", err, output)
	}
	if strings.TrimSpace(toString(payload["generated_at"])) == "" {
		t.Fatalf("expected generated_at in json output, got: %v", payload)
	}
	runtimeMetrics, ok := payload["runtime_metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected runtime_metrics object, got: %#v", payload["runtime_metrics"])
	}
	tool, ok := runtimeMetrics["tool"].(map[string]any)
	if !ok || toFloat64(tool["total"]) < 1 {
		t.Fatalf("expected tool metrics in json output, got: %#v", runtimeMetrics["tool"])
	}
	channel, ok := runtimeMetrics["channel"].(map[string]any)
	if !ok || toFloat64(channel["send_attempts"]) < 1 {
		t.Fatalf("expected channel metrics in json output, got: %#v", runtimeMetrics["channel"])
	}
	memorySection, ok := runtimeMetrics["memory"].(map[string]any)
	if !ok || toFloat64(memorySection["total_items"]) < 1 {
		t.Fatalf("expected memory metrics in json output, got: %#v", runtimeMetrics["memory"])
	}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
