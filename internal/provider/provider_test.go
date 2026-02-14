package provider

import (
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/auth"
	"github.com/MEKXH/golem/internal/config"
)

func TestNewChatModel_NoProvider(t *testing.T) {
	cfg := config.DefaultConfig()

	_, err := NewChatModel(nil, cfg)
	if err == nil {
		t.Error("expected error when no provider configured")
	}
}

func TestProviderFromModel(t *testing.T) {
	tests := []struct {
		model string
		want  providerName
	}{
		{model: "openai/gpt-4o", want: providerOpenAI},
		{model: "anthropic/claude-sonnet-4-5", want: providerClaude},
		{model: "claude/claude-3-5-sonnet", want: providerClaude},
		{model: "gemini/gemini-2.0-flash", want: providerGemini},
		{model: "ollama/llama3.1", want: providerOllama},
		{model: "unknown/model", want: ""},
		{model: "no-prefix-model", want: ""},
	}

	for _, tt := range tests {
		if got := providerFromModel(tt.model); got != tt.want {
			t.Fatalf("providerFromModel(%q)=%q want %q", tt.model, got, tt.want)
		}
	}
}

func TestResolveProvider_PrefersModelMappedProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "openai/gpt-4o"
	cfg.Providers.OpenRouter.APIKey = "openrouter-key"
	cfg.Providers.OpenAI.APIKey = "openai-key"

	got, _, err := resolveProvider(cfg)
	if err != nil {
		t.Fatalf("resolveProvider returned error: %v", err)
	}
	if got != providerOpenAI {
		t.Fatalf("expected provider %q, got %q", providerOpenAI, got)
	}
}

func TestResolveProvider_FallbackOrder(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "no-prefix-model"
	cfg.Providers.Qwen.APIKey = "qwen-key"
	cfg.Providers.DeepSeek.APIKey = "deepseek-key"

	got, _, err := resolveProvider(cfg)
	if err != nil {
		t.Fatalf("resolveProvider returned error: %v", err)
	}
	if got != providerDeepSeek {
		t.Fatalf("expected provider %q, got %q", providerDeepSeek, got)
	}
}

func TestResolveProvider_SupportsArkAndQianfan(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "ark/my-model"
	cfg.Providers.Ark.APIKey = "ark-key"
	cfg.Providers.Ark.BaseURL = "https://ark.example.com/v1"

	got, _, err := resolveProvider(cfg)
	if err != nil {
		t.Fatalf("resolveProvider(ark) returned error: %v", err)
	}
	if got != providerArk {
		t.Fatalf("expected provider %q, got %q", providerArk, got)
	}

	cfg = config.DefaultConfig()
	cfg.Agents.Defaults.Model = "qianfan/ernie"
	cfg.Providers.Qianfan.APIKey = "qianfan-key"
	cfg.Providers.Qianfan.BaseURL = "https://qianfan.example.com/v1"

	got, _, err = resolveProvider(cfg)
	if err != nil {
		t.Fatalf("resolveProvider(qianfan) returned error: %v", err)
	}
	if got != providerQianfan {
		t.Fatalf("expected provider %q, got %q", providerQianfan, got)
	}
}

func TestResolveProvider_OllamaRequiresBaseURL(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "ollama/llama3.1"
	cfg.Providers.Ollama.BaseURL = ""

	if _, _, err := resolveProvider(cfg); err == nil {
		t.Fatal("expected resolveProvider to fail when ollama base_url is empty")
	}
}

func TestResolveProvider_UsesStoredAuthTokenWhenAPIKeyMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := auth.SetCredential("openai", &auth.Credential{
		AccessToken: "auth-openai-token",
		Provider:    "openai",
		AuthMethod:  "token",
	}); err != nil {
		t.Fatalf("auth.SetCredential: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "openai/gpt-4o"
	cfg.Providers.OpenAI.APIKey = ""

	got, pcfg, err := resolveProvider(cfg)
	if err != nil {
		t.Fatalf("resolveProvider returned error: %v", err)
	}
	if got != providerOpenAI {
		t.Fatalf("expected provider %q, got %q", providerOpenAI, got)
	}
	if pcfg.APIKey != "auth-openai-token" {
		t.Fatalf("expected auth token injected as api key, got %q", pcfg.APIKey)
	}
}

func TestResolveProvider_RefreshesOAuthCredentialWhenNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := auth.SetCredential("openai", &auth.Credential{
		AccessToken:  "stale-token",
		RefreshToken: "refresh-token",
		Provider:     "openai",
		AuthMethod:   "oauth",
		ExpiresAt:    time.Now().Add(30 * time.Second),
	}); err != nil {
		t.Fatalf("auth.SetCredential: %v", err)
	}

	orig := refreshProviderCredential
	refreshProviderCredential = func(name providerName, cred *auth.Credential) (*auth.Credential, error) {
		return &auth.Credential{
			AccessToken:  "fresh-token",
			RefreshToken: cred.RefreshToken,
			Provider:     cred.Provider,
			AuthMethod:   "oauth",
			ExpiresAt:    time.Now().Add(2 * time.Hour),
		}, nil
	}
	defer func() { refreshProviderCredential = orig }()

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "openai/gpt-4o"
	cfg.Providers.OpenAI.APIKey = ""

	got, pcfg, err := resolveProvider(cfg)
	if err != nil {
		t.Fatalf("resolveProvider returned error: %v", err)
	}
	if got != providerOpenAI {
		t.Fatalf("expected provider %q, got %q", providerOpenAI, got)
	}
	if pcfg.APIKey != "fresh-token" {
		t.Fatalf("expected refreshed token injected, got %q", pcfg.APIKey)
	}
}
