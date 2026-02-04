package commands

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/MEKXH/golem/internal/config"
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
