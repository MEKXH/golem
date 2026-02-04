package commands

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/MEKXH/golem/internal/config"
)

func TestInitCommand_CreatesConfigAndWorkspace(t *testing.T) {
    tmpDir := t.TempDir()
    t.Setenv("HOME", tmpDir)
    t.Setenv("USERPROFILE", tmpDir)

    if err := runInit(nil, nil); err != nil {
        t.Fatalf("runInit error: %v", err)
    }

    configPath := config.ConfigPath()
    if _, err := os.Stat(configPath); err != nil {
        t.Fatalf("expected config file at %s: %v", configPath, err)
    }

    cfg := config.DefaultConfig()
    if _, err := os.Stat(cfg.WorkspacePath()); err != nil {
        t.Fatalf("expected workspace dir at %s: %v", cfg.WorkspacePath(), err)
    }

    identityPath := filepath.Join(cfg.WorkspacePath(), "IDENTITY.md")
    if _, err := os.Stat(identityPath); err != nil {
        t.Fatalf("expected identity file at %s: %v", identityPath, err)
    }
}
