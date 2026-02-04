package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestDefaultConfig(t *testing.T) {
    cfg := DefaultConfig()

    if cfg.Agents.Defaults.MaxToolIterations != 20 {
        t.Errorf("expected MaxToolIterations=20, got %d", cfg.Agents.Defaults.MaxToolIterations)
    }
    if cfg.Agents.Defaults.Temperature != 0.7 {
        t.Errorf("expected Temperature=0.7, got %f", cfg.Agents.Defaults.Temperature)
    }
    if cfg.Gateway.Port != 18790 {
        t.Errorf("expected Port=18790, got %d", cfg.Gateway.Port)
    }
}

func TestLoadConfig_CreatesDefault(t *testing.T) {
    tmpDir := t.TempDir()
    t.Setenv("HOME", tmpDir)
    t.Setenv("USERPROFILE", tmpDir)

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load() error: %v", err)
    }
    if cfg.Agents.Defaults.MaxToolIterations != 20 {
        t.Errorf("expected default MaxToolIterations=20, got %d", cfg.Agents.Defaults.MaxToolIterations)
    }
}

func TestLoadConfig_PascalCaseKeys(t *testing.T) {
    tmpDir := t.TempDir()
    t.Setenv("HOME", tmpDir)
    t.Setenv("USERPROFILE", tmpDir)

    configPath := ConfigPath()
    if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
        t.Fatalf("MkdirAll: %v", err)
    }

    raw := `{
  "Providers": {
    "OpenRouter": {
      "APIKey": "test-key",
      "BaseURL": "http://example.test/v1"
    }
  }
}`

    if err := os.WriteFile(configPath, []byte(raw), 0644); err != nil {
        t.Fatalf("WriteFile: %v", err)
    }

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load() error: %v", err)
    }

    if cfg.Providers.OpenRouter.APIKey != "test-key" {
        t.Fatalf("expected APIKey loaded, got %q", cfg.Providers.OpenRouter.APIKey)
    }
    if cfg.Providers.OpenRouter.BaseURL != "http://example.test/v1" {
        t.Fatalf("expected BaseURL loaded, got %q", cfg.Providers.OpenRouter.BaseURL)
    }
}
