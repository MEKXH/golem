package provider

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/MEKXH/golem/internal/config"
)

// NewChatModel creates a ChatModel based on configuration
func NewChatModel(ctx context.Context, cfg *config.Config) (model.ChatModel, error) {
    p := cfg.Providers
    d := cfg.Agents.Defaults

    switch {
    case p.OpenRouter.APIKey != "":
        return newOpenRouterModel(ctx, p.OpenRouter, d)
    case p.Claude.APIKey != "":
        return newClaudeModel(ctx, p.Claude, d)
    case p.OpenAI.APIKey != "":
        return newOpenAIModel(ctx, p.OpenAI, d)
    case p.DeepSeek.APIKey != "":
        return newDeepSeekModel(ctx, p.DeepSeek, d)
    case p.Ollama.BaseURL != "":
        return newOllamaModel(ctx, p.Ollama, d)
    default:
        return nil, fmt.Errorf("no provider configured: set api_key for at least one provider")
    }
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
