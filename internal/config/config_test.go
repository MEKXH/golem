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
	if !cfg.Tools.Exec.RestrictToWorkspace {
		t.Errorf("expected RestrictToWorkspace=true by default")
	}
	if cfg.Tools.Voice.Enabled {
		t.Errorf("expected voice transcription disabled by default")
	}
	if cfg.Tools.Voice.Provider != "openai" {
		t.Errorf("expected voice provider=openai, got %q", cfg.Tools.Voice.Provider)
	}
	if cfg.Tools.Voice.TimeoutSeconds != 30 {
		t.Errorf("expected voice timeout_seconds=30, got %d", cfg.Tools.Voice.TimeoutSeconds)
	}
	if !cfg.Heartbeat.Enabled {
		t.Errorf("expected heartbeat enabled by default")
	}
	if cfg.Heartbeat.Interval != 30 {
		t.Errorf("expected heartbeat interval=30, got %d", cfg.Heartbeat.Interval)
	}
	if cfg.Heartbeat.MaxIdleMinutes != 720 {
		t.Errorf("expected heartbeat max_idle_minutes=720, got %d", cfg.Heartbeat.MaxIdleMinutes)
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

func TestWorkspacePath_Default(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents.Defaults.Workspace = ""
	cfg.Agents.Defaults.WorkspaceMode = "default"
	got := cfg.WorkspacePath()
	want := filepath.Join(ConfigDir(), "workspace")
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestWorkspacePath_PathModeRequiresWorkspace(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents.Defaults.WorkspaceMode = "path"
	cfg.Agents.Defaults.Workspace = ""
	if _, err := cfg.WorkspacePathChecked(); err == nil {
		t.Fatal("expected error")
	}
}

func TestWorkspacePath_CwdModeUsesCwd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents.Defaults.WorkspaceMode = "cwd"
	got, err := cfg.WorkspacePathChecked()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if got != wd {
		t.Fatalf("got %s want %s", got, wd)
	}
}

func TestLoadConfig_SaveFailureReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	// Create a file where ConfigDir() expects a directory.
	badConfigDir := filepath.Join(tmpDir, ".golem")
	if err := os.WriteFile(badConfigDir, []byte("not-a-dir"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to return error when default config cannot be saved")
	}
}

func TestValidate_GatewayPortRange(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Gateway.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for gateway.port=0")
	}

	cfg = DefaultConfig()
	cfg.Gateway.Port = 65536
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for gateway.port>65535")
	}
}

func TestValidate_LogLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Log.Level = "DEBUG"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected uppercase log level to be normalized, got error: %v", err)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("expected normalized log level debug, got %q", cfg.Log.Level)
	}

	cfg = DefaultConfig()
	cfg.Log.Level = "invalid-level"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid log level")
	}
}

func TestValidate_HeartbeatDefaultsAndClamp(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Heartbeat.Interval = 0
	cfg.Heartbeat.MaxIdleMinutes = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Heartbeat.Interval != 30 {
		t.Fatalf("expected heartbeat interval defaulted to 30, got %d", cfg.Heartbeat.Interval)
	}
	if cfg.Heartbeat.MaxIdleMinutes != 720 {
		t.Fatalf("expected heartbeat max_idle_minutes defaulted to 720, got %d", cfg.Heartbeat.MaxIdleMinutes)
	}

	cfg = DefaultConfig()
	cfg.Heartbeat.Interval = 1
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error when clamping heartbeat interval: %v", err)
	}
	if cfg.Heartbeat.Interval != 5 {
		t.Fatalf("expected heartbeat interval clamped to 5, got %d", cfg.Heartbeat.Interval)
	}
}

func TestValidate_HeartbeatNegativeValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Heartbeat.Interval = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for heartbeat.interval < 0")
	}

	cfg = DefaultConfig()
	cfg.Heartbeat.MaxIdleMinutes = -10
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for heartbeat.max_idle_minutes < 0")
	}
}

func TestValidate_VoiceDefaultsAndProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tools.Voice.Provider = ""
	cfg.Tools.Voice.TimeoutSeconds = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Tools.Voice.Provider != "openai" {
		t.Fatalf("expected provider default openai, got %q", cfg.Tools.Voice.Provider)
	}
	if cfg.Tools.Voice.TimeoutSeconds != 30 {
		t.Fatalf("expected timeout default 30, got %d", cfg.Tools.Voice.TimeoutSeconds)
	}

	cfg = DefaultConfig()
	cfg.Tools.Voice.Enabled = true
	cfg.Tools.Voice.Provider = "unknown"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for unsupported voice provider")
	}
}

func TestValidate_VoiceNegativeTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tools.Voice.TimeoutSeconds = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for voice timeout < 0")
	}
}

func TestDefaultConfig_PolicyDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Policy.Mode != "strict" {
		t.Fatalf("expected policy mode strict, got %q", cfg.Policy.Mode)
	}
	if cfg.Policy.OffTTL != "" {
		t.Fatalf("expected empty policy off_ttl by default, got %q", cfg.Policy.OffTTL)
	}
	if cfg.Policy.AllowPersistentOff {
		t.Fatalf("expected allow_persistent_off=false by default")
	}
	if len(cfg.Policy.RequireApproval) != 0 {
		t.Fatalf("expected empty require_approval by default, got %v", cfg.Policy.RequireApproval)
	}
}

func TestValidate_PolicyMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Policy.Mode = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid policy mode")
	}
}

func TestValidate_PersistentOffGate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = ""
	cfg.Policy.AllowPersistentOff = false
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for persistent off without allow_persistent_off")
	}

	cfg = DefaultConfig()
	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = "30m"
	cfg.Policy.AllowPersistentOff = false
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected ttl-based off mode to be valid, got error: %v", err)
	}

	cfg = DefaultConfig()
	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = ""
	cfg.Policy.AllowPersistentOff = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected persistent off to be allowed when gated, got error: %v", err)
	}

	cfg = DefaultConfig()
	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = "foo"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid policy.off_ttl duration")
	}

	cfg = DefaultConfig()
	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = "-10s"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for non-positive policy.off_ttl")
	}
}

func TestValidate_MCPServers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MCP.Servers = map[string]MCPServerConfig{
		"bad_transport": {Transport: "tcp"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid MCP transport")
	}

	cfg = DefaultConfig()
	cfg.MCP.Servers = map[string]MCPServerConfig{
		"stdio_missing_command": {Transport: "stdio"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for stdio server without command")
	}

	cfg = DefaultConfig()
	cfg.MCP.Servers = map[string]MCPServerConfig{
		"http_missing_url": {Transport: "http_sse"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for http_sse server without url")
	}

	cfg = DefaultConfig()
	cfg.MCP.Servers = map[string]MCPServerConfig{
		" localfs ": {
			Transport: "stdio",
			Command:   "npx",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for MCP server name with surrounding whitespace")
	}

	cfg = DefaultConfig()
	cfg.MCP.Servers = map[string]MCPServerConfig{
		"stdio_ok": {
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "some-mcp-server"},
		},
		"http_ok": {
			Transport: "http_sse",
			URL:       "http://localhost:8080/sse",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid MCP servers config, got error: %v", err)
	}

	disabled := false
	cfg = DefaultConfig()
	cfg.MCP.Servers = map[string]MCPServerConfig{
		"disabled_invalid": {
			Enabled:   &disabled,
			Transport: "invalid",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected disabled MCP server to skip transport validation, got error: %v", err)
	}
}
