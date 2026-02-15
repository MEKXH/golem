package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Config root configuration
type Config struct {
	Agents    AgentsConfig    `mapstructure:"agents"`
	Channels  ChannelsConfig  `mapstructure:"channels"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Gateway   GatewayConfig   `mapstructure:"gateway"`
	Log       LogConfig       `mapstructure:"log"`
	Policy    PolicyConfig    `mapstructure:"policy"`
	MCP       MCPConfig       `mapstructure:"mcp"`
	Tools     ToolsConfig     `mapstructure:"tools"`
	Heartbeat HeartbeatConfig `mapstructure:"heartbeat"`
}

// PolicyConfig runtime policy settings.
type PolicyConfig struct {
	Mode               string   `mapstructure:"mode"`
	OffTTL             string   `mapstructure:"off_ttl"`
	AllowPersistentOff bool     `mapstructure:"allow_persistent_off"`
	RequireApproval    []string `mapstructure:"require_approval"`
}

// MCPConfig MCP server settings.
type MCPConfig struct {
	Servers map[string]MCPServerConfig `mapstructure:"servers"`
}

// MCPServerConfig MCP server transport settings.
type MCPServerConfig struct {
	Enabled   *bool             `mapstructure:"enabled"`
	Transport string            `mapstructure:"transport"`
	Command   string            `mapstructure:"command"`
	Args      []string          `mapstructure:"args"`
	Env       map[string]string `mapstructure:"env"`
	URL       string            `mapstructure:"url"`
	Headers   map[string]string `mapstructure:"headers"`
}

// IsMCPServerEnabled returns true unless the server is explicitly disabled.
func IsMCPServerEnabled(server MCPServerConfig) bool {
	if server.Enabled == nil {
		return true
	}
	return *server.Enabled
}

// AgentsConfig agent settings
type AgentsConfig struct {
	Defaults AgentDefaults         `mapstructure:"defaults"`
	Subagent SubagentRuntimeConfig `mapstructure:"subagent"`
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

// SubagentRuntimeConfig controls delegated subagent execution policy.
type SubagentRuntimeConfig struct {
	TimeoutSeconds int `mapstructure:"timeout_seconds"`
	Retry          int `mapstructure:"retry"`
	MaxConcurrency int `mapstructure:"max_concurrency"`
}

// ChannelsConfig channel settings
type ChannelsConfig struct {
	Telegram TelegramConfig        `mapstructure:"telegram"`
	WhatsApp WhatsAppConfig        `mapstructure:"whatsapp"`
	Feishu   FeishuConfig          `mapstructure:"feishu"`
	Discord  DiscordConfig         `mapstructure:"discord"`
	Slack    SlackConfig           `mapstructure:"slack"`
	QQ       QQConfig              `mapstructure:"qq"`
	DingTalk DingTalkConfig        `mapstructure:"dingtalk"`
	MaixCam  MaixCamConfig         `mapstructure:"maixcam"`
	Outbound ChannelOutboundConfig `mapstructure:"outbound"`
}

// ChannelOutboundConfig controls outbound reliability behavior.
type ChannelOutboundConfig struct {
	MaxConcurrentSends int `mapstructure:"max_concurrent_sends"`
	RetryMaxAttempts   int `mapstructure:"retry_max_attempts"`
	RetryBaseBackoffMs int `mapstructure:"retry_base_backoff_ms"`
	RetryMaxBackoffMs  int `mapstructure:"retry_max_backoff_ms"`
	RateLimitPerSecond int `mapstructure:"rate_limit_per_second"`
	DedupWindowSeconds int `mapstructure:"dedup_window_seconds"`
}

// TelegramConfig telegram bot settings
type TelegramConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Token     string   `mapstructure:"token"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// WhatsAppConfig WhatsApp bridge settings
type WhatsAppConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	BridgeURL string   `mapstructure:"bridge_url"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// FeishuConfig Feishu bot settings
type FeishuConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	AppID             string   `mapstructure:"app_id"`
	AppSecret         string   `mapstructure:"app_secret"`
	EncryptKey        string   `mapstructure:"encrypt_key"`
	VerificationToken string   `mapstructure:"verification_token"`
	AllowFrom         []string `mapstructure:"allow_from"`
}

// DiscordConfig Discord bot settings
type DiscordConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Token     string   `mapstructure:"token"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// SlackConfig Slack bot settings
type SlackConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	BotToken  string   `mapstructure:"bot_token"`
	AppToken  string   `mapstructure:"app_token"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// QQConfig QQ bot settings
type QQConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	AppID     string   `mapstructure:"app_id"`
	AppSecret string   `mapstructure:"app_secret"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// DingTalkConfig DingTalk stream mode settings
type DingTalkConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	AllowFrom    []string `mapstructure:"allow_from"`
}

// MaixCamConfig MaixCam bridge settings
type MaixCamConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Host      string   `mapstructure:"host"`
	Port      int      `mapstructure:"port"`
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

// LogConfig application logging settings
type LogConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// ToolsConfig tool settings
type ToolsConfig struct {
	Web   WebToolsConfig  `mapstructure:"web"`
	Exec  ExecToolConfig  `mapstructure:"exec"`
	Voice VoiceToolConfig `mapstructure:"voice"`
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

// VoiceToolConfig speech-to-text settings for inbound audio.
type VoiceToolConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Provider       string `mapstructure:"provider"`
	Model          string `mapstructure:"model"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

// HeartbeatConfig heartbeat service settings.
type HeartbeatConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	Interval       int  `mapstructure:"interval"`         // minutes
	MaxIdleMinutes int  `mapstructure:"max_idle_minutes"` // minutes
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
			Subagent: SubagentRuntimeConfig{
				TimeoutSeconds: 300,
				Retry:          1,
				MaxConcurrency: 3,
			},
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Enabled:   false,
				AllowFrom: []string{},
			},
			WhatsApp: WhatsAppConfig{
				Enabled:   false,
				AllowFrom: []string{},
			},
			Feishu: FeishuConfig{
				Enabled:   false,
				AllowFrom: []string{},
			},
			Discord: DiscordConfig{
				Enabled:   false,
				AllowFrom: []string{},
			},
			Slack: SlackConfig{
				Enabled:   false,
				AllowFrom: []string{},
			},
			QQ: QQConfig{
				Enabled:   false,
				AllowFrom: []string{},
			},
			DingTalk: DingTalkConfig{
				Enabled:   false,
				AllowFrom: []string{},
			},
			MaixCam: MaixCamConfig{
				Enabled:   false,
				Host:      "0.0.0.0",
				Port:      9000,
				AllowFrom: []string{},
			},
			Outbound: ChannelOutboundConfig{
				MaxConcurrentSends: 16,
				RetryMaxAttempts:   3,
				RetryBaseBackoffMs: 200,
				RetryMaxBackoffMs:  2000,
				RateLimitPerSecond: 20,
				DedupWindowSeconds: 30,
			},
		},
		Providers: ProvidersConfig{},
		Gateway: GatewayConfig{
			Host:  "0.0.0.0",
			Port:  18790,
			Token: "",
		},
		Log: LogConfig{
			Level: "info",
			File:  "",
		},
		Policy: PolicyConfig{
			Mode:               "strict",
			OffTTL:             "",
			AllowPersistentOff: false,
			RequireApproval:    []string{},
		},
		MCP: MCPConfig{
			Servers: map[string]MCPServerConfig{},
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
			Voice: VoiceToolConfig{
				Enabled:        false,
				Provider:       "openai",
				Model:          "gpt-4o-mini-transcribe",
				TimeoutSeconds: 30,
			},
		},
		Heartbeat: HeartbeatConfig{
			Enabled:        true,
			Interval:       30,
			MaxIdleMinutes: 720,
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

	if c.Agents.Subagent.TimeoutSeconds < 0 {
		return fmt.Errorf("agents.subagent.timeout_seconds must not be negative, got %d", c.Agents.Subagent.TimeoutSeconds)
	}
	if c.Agents.Subagent.TimeoutSeconds == 0 {
		c.Agents.Subagent.TimeoutSeconds = 300
	}
	if c.Agents.Subagent.Retry < 0 {
		return fmt.Errorf("agents.subagent.retry must not be negative, got %d", c.Agents.Subagent.Retry)
	}
	if c.Agents.Subagent.Retry == 0 {
		c.Agents.Subagent.Retry = 1
	}
	if c.Agents.Subagent.MaxConcurrency < 0 {
		return fmt.Errorf("agents.subagent.max_concurrency must not be negative, got %d", c.Agents.Subagent.MaxConcurrency)
	}
	if c.Agents.Subagent.MaxConcurrency == 0 {
		c.Agents.Subagent.MaxConcurrency = 3
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
	if c.Channels.MaixCam.Port != 0 && (c.Channels.MaixCam.Port < 1 || c.Channels.MaixCam.Port > 65535) {
		return fmt.Errorf("channels.maixcam.port must be between 1 and 65535, got %d", c.Channels.MaixCam.Port)
	}

	if c.Channels.Outbound.MaxConcurrentSends < 0 {
		return fmt.Errorf("channels.outbound.max_concurrent_sends must not be negative, got %d", c.Channels.Outbound.MaxConcurrentSends)
	}
	if c.Channels.Outbound.MaxConcurrentSends == 0 {
		c.Channels.Outbound.MaxConcurrentSends = 16
	}
	if c.Channels.Outbound.RetryMaxAttempts < 0 {
		return fmt.Errorf("channels.outbound.retry_max_attempts must not be negative, got %d", c.Channels.Outbound.RetryMaxAttempts)
	}
	if c.Channels.Outbound.RetryMaxAttempts == 0 {
		c.Channels.Outbound.RetryMaxAttempts = 3
	}
	if c.Channels.Outbound.RetryBaseBackoffMs < 0 {
		return fmt.Errorf("channels.outbound.retry_base_backoff_ms must not be negative, got %d", c.Channels.Outbound.RetryBaseBackoffMs)
	}
	if c.Channels.Outbound.RetryBaseBackoffMs == 0 {
		c.Channels.Outbound.RetryBaseBackoffMs = 200
	}
	if c.Channels.Outbound.RetryMaxBackoffMs < 0 {
		return fmt.Errorf("channels.outbound.retry_max_backoff_ms must not be negative, got %d", c.Channels.Outbound.RetryMaxBackoffMs)
	}
	if c.Channels.Outbound.RetryMaxBackoffMs == 0 {
		c.Channels.Outbound.RetryMaxBackoffMs = 2000
	}
	if c.Channels.Outbound.RetryMaxBackoffMs < c.Channels.Outbound.RetryBaseBackoffMs {
		c.Channels.Outbound.RetryMaxBackoffMs = c.Channels.Outbound.RetryBaseBackoffMs
	}
	if c.Channels.Outbound.RateLimitPerSecond < 0 {
		return fmt.Errorf("channels.outbound.rate_limit_per_second must not be negative, got %d", c.Channels.Outbound.RateLimitPerSecond)
	}
	if c.Channels.Outbound.RateLimitPerSecond == 0 {
		c.Channels.Outbound.RateLimitPerSecond = 20
	}
	if c.Channels.Outbound.DedupWindowSeconds < 0 {
		return fmt.Errorf("channels.outbound.dedup_window_seconds must not be negative, got %d", c.Channels.Outbound.DedupWindowSeconds)
	}
	if c.Channels.Outbound.DedupWindowSeconds == 0 {
		c.Channels.Outbound.DedupWindowSeconds = 30
	}

	level := strings.ToLower(strings.TrimSpace(c.Log.Level))
	if level == "" {
		c.Log.Level = "info"
	} else {
		validLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}
		if !validLevels[level] {
			return fmt.Errorf("log.level must be one of debug, info, warn, error; got %q", c.Log.Level)
		}
		c.Log.Level = level
	}

	policyMode := strings.ToLower(strings.TrimSpace(c.Policy.Mode))
	if policyMode == "" {
		policyMode = "strict"
	}
	switch policyMode {
	case "strict", "relaxed", "off":
	default:
		return fmt.Errorf("policy.mode must be one of strict, relaxed, off; got %q", c.Policy.Mode)
	}
	c.Policy.Mode = policyMode

	offTTL := strings.TrimSpace(c.Policy.OffTTL)
	if offTTL != "" {
		offDuration, err := time.ParseDuration(offTTL)
		if err != nil {
			return fmt.Errorf("policy.off_ttl must be a valid duration, got %q: %w", c.Policy.OffTTL, err)
		}
		if offDuration <= 0 {
			return fmt.Errorf("policy.off_ttl must be > 0, got %q", c.Policy.OffTTL)
		}
		c.Policy.OffTTL = offTTL
	}
	if c.Policy.Mode == "off" && offTTL == "" && !c.Policy.AllowPersistentOff {
		return fmt.Errorf("policy.mode=off without policy.off_ttl requires policy.allow_persistent_off=true")
	}

	for serverName, server := range c.MCP.Servers {
		name := strings.TrimSpace(serverName)
		if name == "" {
			return fmt.Errorf("mcp.servers contains an empty server name")
		}
		if name != serverName {
			return fmt.Errorf("mcp.servers.%q has leading or trailing whitespace; use %q", serverName, name)
		}

		if !IsMCPServerEnabled(server) {
			transport := strings.TrimSpace(server.Transport)
			if transport != "" {
				server.Transport = strings.ToLower(transport)
				c.MCP.Servers[serverName] = server
			}
			continue
		}

		transport := strings.ToLower(strings.TrimSpace(server.Transport))
		switch transport {
		case "stdio":
			if strings.TrimSpace(server.Command) == "" {
				return fmt.Errorf("mcp.servers.%s.command is required when transport=stdio", serverName)
			}
		case "http_sse":
			if strings.TrimSpace(server.URL) == "" {
				return fmt.Errorf("mcp.servers.%s.url is required when transport=http_sse", serverName)
			}
		default:
			return fmt.Errorf("mcp.servers.%s.transport must be one of stdio, http_sse; got %q", serverName, server.Transport)
		}

		server.Transport = transport
		c.MCP.Servers[serverName] = server
	}

	if c.Heartbeat.Interval < 0 {
		return fmt.Errorf("heartbeat.interval must not be negative, got %d", c.Heartbeat.Interval)
	}
	if c.Heartbeat.Interval == 0 {
		c.Heartbeat.Interval = 30
	}
	if c.Heartbeat.Interval > 0 && c.Heartbeat.Interval < 5 {
		c.Heartbeat.Interval = 5
	}

	if c.Heartbeat.MaxIdleMinutes < 0 {
		return fmt.Errorf("heartbeat.max_idle_minutes must not be negative, got %d", c.Heartbeat.MaxIdleMinutes)
	}
	if c.Heartbeat.MaxIdleMinutes == 0 {
		c.Heartbeat.MaxIdleMinutes = 720
	}

	voiceProvider := strings.ToLower(strings.TrimSpace(c.Tools.Voice.Provider))
	if voiceProvider == "" {
		voiceProvider = "openai"
	}
	if c.Tools.Voice.Enabled && voiceProvider != "openai" {
		return fmt.Errorf("tools.voice.provider must be \"openai\" when enabled; got %q", c.Tools.Voice.Provider)
	}
	c.Tools.Voice.Provider = voiceProvider

	if c.Tools.Voice.TimeoutSeconds < 0 {
		return fmt.Errorf("tools.voice.timeout_seconds must not be negative, got %d", c.Tools.Voice.TimeoutSeconds)
	}
	if c.Tools.Voice.TimeoutSeconds == 0 {
		c.Tools.Voice.TimeoutSeconds = 30
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
