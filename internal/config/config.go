package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Config root configuration
type Config struct {
	Agents    AgentsConfig    `mapstructure:"agents"`
	Channels  ChannelsConfig  `mapstructure:"channels"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Gateway   GatewayConfig   `mapstructure:"gateway"`
	Tools     ToolsConfig     `mapstructure:"tools"`
}

// AgentsConfig agent settings
type AgentsConfig struct {
	Defaults AgentDefaults `mapstructure:"defaults"`
}

// AgentDefaults default agent parameters
type AgentDefaults struct {
	Workspace         string  `mapstructure:"workspace"`
	WorkspaceMode     string  `mapstructure:"workspace_mode"`
	Model             string  `mapstructure:"model"`
	MaxTokens         int     `mapstructure:"max_tokens"`
	Temperature       float64 `mapstructure:"temperature"`
	MaxToolIterations int     `mapstructure:"max_tool_iterations"`
}

// ChannelsConfig channel settings
type ChannelsConfig struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
}

// TelegramConfig telegram bot settings
type TelegramConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Token     string   `mapstructure:"token"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// ProvidersConfig LLM provider settings
type ProvidersConfig struct {
	OpenRouter ProviderConfig `mapstructure:"openrouter"`
	Claude     ProviderConfig `mapstructure:"claude"`
	OpenAI     ProviderConfig `mapstructure:"openai"`
	DeepSeek   ProviderConfig `mapstructure:"deepseek"`
	Gemini     ProviderConfig `mapstructure:"gemini"`
	Ark        ProviderConfig `mapstructure:"ark"`
	Qianfan    ProviderConfig `mapstructure:"qianfan"`
	Qwen       ProviderConfig `mapstructure:"qwen"`
	Ollama     ProviderConfig `mapstructure:"ollama"`
}

// ProviderConfig single provider settings
type ProviderConfig struct {
	APIKey    string `mapstructure:"api_key"`
	SecretKey string `mapstructure:"secret_key"`
	BaseURL   string `mapstructure:"base_url"`
}

// GatewayConfig server settings
type GatewayConfig struct {
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
	Token string `mapstructure:"token"`
}

// ToolsConfig tool settings
type ToolsConfig struct {
	Web  WebToolsConfig `mapstructure:"web"`
	Exec ExecToolConfig `mapstructure:"exec"`
}

// WebToolsConfig web tool settings
type WebToolsConfig struct {
	Search WebSearchConfig `mapstructure:"search"`
}

// WebSearchConfig brave search settings
type WebSearchConfig struct {
	APIKey     string `mapstructure:"api_key"`
	MaxResults int    `mapstructure:"max_results"`
}

// ExecToolConfig shell exec settings
type ExecToolConfig struct {
	Timeout             int  `mapstructure:"timeout"`
	RestrictToWorkspace bool `mapstructure:"restrict_to_workspace"`
}

// DefaultConfig returns config with sensible defaults
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("failed to resolve home directory, using current directory as fallback", "error", err)
		homeDir = "."
	}
	return &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:         filepath.Join(homeDir, ".golem", "workspace"),
				WorkspaceMode:     "default",
				Model:             "anthropic/claude-sonnet-4-5",
				MaxTokens:         8192,
				Temperature:       0.7,
				MaxToolIterations: 20,
			},
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Enabled:   false,
				AllowFrom: []string{},
			},
		},
		Providers: ProvidersConfig{},
		Gateway: GatewayConfig{
			Host:  "0.0.0.0",
			Port:  18790,
			Token: "",
		},
		Tools: ToolsConfig{
			Web: WebToolsConfig{
				Search: WebSearchConfig{
					MaxResults: 5,
				},
			},
			Exec: ExecToolConfig{
				Timeout:             60,
				RestrictToWorkspace: true,
			},
		},
	}
}

// ConfigDir returns the golem config directory
func ConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".golem")
}

// ConfigPath returns the config file path
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

// Load loads config from file or returns defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	configPath := ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := Save(cfg); err != nil {
			return cfg, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("json")
	v.SetEnvPrefix("GOLEM")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return cfg, err
	}

	if err := v.Unmarshal(cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "mapstructure"
		dc.MatchName = func(mapKey, fieldName string) bool {
			return normalizeKey(mapKey) == normalizeKey(fieldName)
		}
	}); err != nil {
		return cfg, err
	}

	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func normalizeKey(input string) string {
	input = strings.ReplaceAll(input, "_", "")
	input = strings.ReplaceAll(input, "-", "")
	return strings.ToLower(input)
}

// Save saves config to file
func Save(cfg *Config) error {
	configPath := ConfigPath()

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// Validate checks that the configuration values are within acceptable ranges.
func (c *Config) Validate() error {
	d := &c.Agents.Defaults

	if d.MaxToolIterations < 0 {
		return fmt.Errorf("agents.defaults.max_tool_iterations must not be negative, got %d", d.MaxToolIterations)
	}
	if d.MaxToolIterations == 0 {
		d.MaxToolIterations = 20
	}

	if d.Temperature < 0 || d.Temperature > 2.0 {
		return fmt.Errorf("agents.defaults.temperature must be between 0 and 2.0, got %f", d.Temperature)
	}

	if d.MaxTokens <= 0 {
		return fmt.Errorf("agents.defaults.max_tokens must be > 0, got %d", d.MaxTokens)
	}

	mode := strings.TrimSpace(d.WorkspaceMode)
	if mode != "" {
		validModes := map[string]bool{"default": true, "cwd": true, "path": true}
		if !validModes[strings.ToLower(mode)] {
			return fmt.Errorf("agents.defaults.workspace_mode must be one of: default, cwd, path; got %q", mode)
		}
		if strings.EqualFold(mode, "path") && strings.TrimSpace(d.Workspace) == "" {
			return fmt.Errorf("agents.defaults.workspace must be non-empty when workspace_mode is \"path\"")
		}
	}

	if c.Gateway.Port <= 0 || c.Gateway.Port > 65535 {
		return fmt.Errorf("gateway.port must be between 1 and 65535, got %d", c.Gateway.Port)
	}

	return nil
}

// WorkspacePath returns the expanded workspace path
func (c *Config) WorkspacePath() string {
	path, err := c.WorkspacePathChecked()
	if err != nil {
		return filepath.Join(ConfigDir(), "workspace")
	}
	return path
}

// WorkspacePathChecked returns the expanded workspace path or an error if invalid.
func (c *Config) WorkspacePathChecked() (string, error) {
	mode := strings.TrimSpace(c.Agents.Defaults.WorkspaceMode)
	if mode == "" || strings.EqualFold(mode, "default") {
		return filepath.Join(ConfigDir(), "workspace"), nil
	}
	if strings.EqualFold(mode, "cwd") {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve cwd: %w", err)
		}
		return wd, nil
	}
	if !strings.EqualFold(mode, "path") {
		return "", fmt.Errorf("unknown workspace_mode: %s", mode)
	}
	if c.Agents.Defaults.Workspace == "" {
		return "", fmt.Errorf("workspace is required when workspace_mode=path")
	}
	if len(c.Agents.Defaults.Workspace) > 0 && c.Agents.Defaults.Workspace[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory for workspace path: %w", err)
		}
		rest := c.Agents.Defaults.Workspace[1:]
		rest = strings.TrimPrefix(rest, string(filepath.Separator))
		rest = strings.TrimPrefix(rest, "/")
		return filepath.Join(homeDir, rest), nil
	}
	return c.Agents.Defaults.Workspace, nil
}
