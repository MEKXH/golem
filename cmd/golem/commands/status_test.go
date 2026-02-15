package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/metrics"
)

func TestStatusCommand_PrintsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	output := captureOutput(t, func() {
		if err := runStatus(nil, nil); err != nil {
			t.Fatalf("runStatus error: %v", err)
		}
	})

	if !strings.Contains(output, "Golem Status") {
		t.Fatalf("expected status output, got: %s", output)
	}
	if !strings.Contains(output, "Config:") {
		t.Fatalf("expected config line, got: %s", output)
	}
	if !strings.Contains(output, "Mode: default") {
		t.Fatalf("expected workspace mode line, got: %s", output)
	}
	if !strings.Contains(output, "Tools:") {
		t.Fatalf("expected tools section, got: %s", output)
	}
	if !strings.Contains(output, "Runtime Metrics:") {
		t.Fatalf("expected runtime metrics section, got: %s", output)
	}
	if !strings.Contains(output, "no runtime data yet") {
		t.Fatalf("expected no runtime data message, got: %s", output)
	}
	if !strings.Contains(output, "web_search: enabled") {
		t.Fatalf("expected web_search readiness line, got: %s", output)
	}
	if !strings.Contains(output, "voice_transcription: disabled") {
		t.Fatalf("expected voice transcription status line, got: %s", output)
	}
	if !strings.Contains(output, "edit_file: ready") || !strings.Contains(output, "append_file: ready") {
		t.Fatalf("expected edit/append tool readiness lines, got: %s", output)
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

	output := captureOutput(t, func() {
		if err := runStatus(nil, nil); err != nil {
			t.Fatalf("runStatus error: %v", err)
		}
	})

	if !strings.Contains(output, "tool_total=2") {
		t.Fatalf("expected tool_total in runtime metrics output, got: %s", output)
	}
	if !strings.Contains(output, "tool_timeout_ratio=0.500") {
		t.Fatalf("expected timeout ratio in runtime metrics output, got: %s", output)
	}
	if !strings.Contains(output, "channel_send_failure_ratio=1.000") {
		t.Fatalf("expected channel failure ratio in runtime metrics output, got: %s", output)
	}
}
