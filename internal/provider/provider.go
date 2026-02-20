package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/auth"
	"github.com/MEKXH/golem/internal/config"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

type providerName string

const (
	providerOpenRouter providerName = "openrouter"
	providerClaude     providerName = "claude"
	providerOpenAI     providerName = "openai"
	providerDeepSeek   providerName = "deepseek"
	providerGemini     providerName = "gemini"
	providerArk        providerName = "ark"
	providerQianfan    providerName = "qianfan"
	providerQwen       providerName = "qwen"
	providerOllama     providerName = "ollama"
)

var refreshProviderCredential = func(name providerName, cred *auth.Credential) (*auth.Credential, error) {
	switch name {
	case providerOpenAI:
		return auth.RefreshAccessToken(cred, auth.OpenAIOAuthConfig())
	default:
		return nil, fmt.Errorf("refresh is not supported for provider %s", name)
	}
}

// NewChatModel creates a ChatModel based on configuration
func NewChatModel(ctx context.Context, cfg *config.Config) (model.ChatModel, error) {
	selected, pcfg, err := resolveProvider(cfg)
	if err != nil {
		return nil, err
	}

	d := cfg.Agents.Defaults
	switch selected {
	case providerOpenRouter:
		return newOpenRouterModel(ctx, pcfg, d)
	case providerClaude:
		return newClaudeModel(ctx, pcfg, d)
	case providerOpenAI:
		return newOpenAIModel(ctx, pcfg, d)
	case providerDeepSeek:
		return newDeepSeekModel(ctx, pcfg, d)
	case providerGemini:
		return newGeminiModel(ctx, pcfg, d)
	case providerArk:
		return newArkModel(ctx, pcfg, d)
	case providerQianfan:
		return newQianfanModel(ctx, pcfg, d)
	case providerQwen:
		return newQwenModel(ctx, pcfg, d)
	case providerOllama:
		return newOllamaModel(ctx, pcfg, d)
	default:
		return nil, fmt.Errorf("unsupported provider selected: %s", selected)
	}
}

func resolveProvider(cfg *config.Config) (providerName, config.ProviderConfig, error) {
	p := cfg.Providers

	if byModel := providerFromModel(cfg.Agents.Defaults.Model); byModel != "" {
		if pcfg, ok := providerConfigByName(p, byModel); ok && providerIsConfigured(byModel, pcfg) {
			pcfg = withResolvedProviderToken(byModel, pcfg)
			return byModel, pcfg, nil
		}
	}

	fallbackOrder := []providerName{
		providerOpenRouter,
		providerClaude,
		providerOpenAI,
		providerDeepSeek,
		providerGemini,
		providerArk,
		providerQianfan,
		providerQwen,
		providerOllama,
	}

	for _, name := range fallbackOrder {
		pcfg, ok := providerConfigByName(p, name)
		if !ok {
			continue
		}
		if providerIsConfigured(name, pcfg) {
			pcfg = withResolvedProviderToken(name, pcfg)
			return name, pcfg, nil
		}
	}

	return "", config.ProviderConfig{}, fmt.Errorf("no provider configured: set api_key/base_url for at least one provider")
}

func providerFromModel(model string) providerName {
	prefix := strings.TrimSpace(model)
	if prefix == "" {
		return ""
	}
	idx := strings.Index(prefix, "/")
	if idx <= 0 {
		return ""
	}

	switch strings.ToLower(prefix[:idx]) {
	case "openrouter":
		return providerOpenRouter
	case "anthropic", "claude":
		return providerClaude
	case "openai":
		return providerOpenAI
	case "deepseek":
		return providerDeepSeek
	case "gemini", "google":
		return providerGemini
	case "ark":
		return providerArk
	case "qianfan":
		return providerQianfan
	case "qwen":
		return providerQwen
	case "ollama":
		return providerOllama
	default:
		return ""
	}
}

func providerConfigByName(p config.ProvidersConfig, name providerName) (config.ProviderConfig, bool) {
	switch name {
	case providerOpenRouter:
		return p.OpenRouter, true
	case providerClaude:
		return p.Claude, true
	case providerOpenAI:
		return p.OpenAI, true
	case providerDeepSeek:
		return p.DeepSeek, true
	case providerGemini:
		return p.Gemini, true
	case providerArk:
		return p.Ark, true
	case providerQianfan:
		return p.Qianfan, true
	case providerQwen:
		return p.Qwen, true
	case providerOllama:
		return p.Ollama, true
	default:
		return config.ProviderConfig{}, false
	}
}

func providerIsConfigured(name providerName, p config.ProviderConfig) bool {
	switch name {
	case providerOllama:
		return strings.TrimSpace(p.BaseURL) != ""
	default:
		if strings.TrimSpace(p.APIKey) != "" {
			return true
		}
		cred := lookupCredential(name)
		return cred != nil && strings.TrimSpace(cred.AccessToken) != ""
	}
}

func withResolvedProviderToken(name providerName, p config.ProviderConfig) config.ProviderConfig {
	if name == providerOllama {
		return p
	}
	if strings.TrimSpace(p.APIKey) != "" {
		return p
	}
	cred := lookupCredential(name)
	if cred != nil && strings.TrimSpace(cred.AccessToken) != "" {
		p.APIKey = strings.TrimSpace(cred.AccessToken)
	}
	return p
}

func lookupCredential(name providerName) *auth.Credential {
	keys := []string{string(name)}
	if name == providerClaude {
		keys = append(keys, "anthropic")
	}
	for _, key := range keys {
		cred, err := auth.GetCredential(key)
		if err == nil && cred != nil && strings.TrimSpace(cred.AccessToken) != "" {
			if cred.AuthMethod == "oauth" && cred.NeedsRefresh() && strings.TrimSpace(cred.RefreshToken) != "" {
				if refreshed, err := refreshProviderCredential(name, cred); err == nil && refreshed != nil && strings.TrimSpace(refreshed.AccessToken) != "" {
					refreshed.Provider = key
					if strings.TrimSpace(refreshed.AuthMethod) == "" {
						refreshed.AuthMethod = "oauth"
					}
					_ = auth.SetCredential(key, refreshed)
					cred = refreshed
				}
			}
			return cred
		}
	}
	return nil
}

func newOpenRouterModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       d.Model,
		APIKey:      p.APIKey,
		BaseURL:     "https://openrouter.ai/api/v1",
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	})
}

func newClaudeModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       d.Model,
		APIKey:      p.APIKey,
		BaseURL:     "https://api.anthropic.com/v1",
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	})
}

func newOpenAIModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	cfg := &openai.ChatModelConfig{
		Model:       d.Model,
		APIKey:      p.APIKey,
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	}
	if p.BaseURL != "" {
		cfg.BaseURL = p.BaseURL
	}
	return openai.NewChatModel(ctx, cfg)
}

func newDeepSeekModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       d.Model,
		APIKey:      p.APIKey,
		BaseURL:     "https://api.deepseek.com/v1",
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	})
}

func newGeminiModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       d.Model,
		APIKey:      p.APIKey,
		BaseURL:     baseURL,
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	})
}

func newArkModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	if strings.TrimSpace(p.BaseURL) == "" {
		return nil, fmt.Errorf("ark provider requires base_url")
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       d.Model,
		APIKey:      p.APIKey,
		BaseURL:     p.BaseURL,
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	})
}

func newQianfanModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	if strings.TrimSpace(p.BaseURL) == "" {
		return nil, fmt.Errorf("qianfan provider requires base_url")
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       d.Model,
		APIKey:      p.APIKey,
		BaseURL:     p.BaseURL,
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	})
}

func newQwenModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       d.Model,
		APIKey:      p.APIKey,
		BaseURL:     baseURL,
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	})
}

func newOllamaModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:       d.Model,
		BaseURL:     baseURL + "/v1",
		Temperature: toFloat32Ptr(d.Temperature),
		MaxTokens:   toIntPtr(d.MaxTokens),
	})
}

func toFloat32Ptr(f float64) *float32 {
	v := float32(f)
	return &v
}

func toIntPtr(i int) *int {
	return &i
}
