package commands

import (
    "fmt"
    "os"
    "strings"

    "github.com/MEKXH/golem/internal/config"
    "github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show Golem configuration status",
        RunE:  runStatus,
    }
}

func runStatus(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    workspacePath, err := cfg.WorkspacePathChecked()
    if err != nil {
        return fmt.Errorf("invalid workspace: %w", err)
    }

    fmt.Println("=== Golem Status ===")
    fmt.Println()

    fmt.Printf("Config: %s\n", config.ConfigPath())
    if _, err := os.Stat(config.ConfigPath()); err == nil {
        fmt.Println("  Status: OK")
    } else {
        fmt.Println("  Status: Not found (run 'golem init')")
    }

    fmt.Printf("\nWorkspace: %s\n", workspacePath)
    if _, err := os.Stat(workspacePath); err == nil {
        fmt.Println("  Status: OK")
    } else {
        fmt.Println("  Status: Not found")
    }
    workspaceMode := strings.TrimSpace(cfg.Agents.Defaults.WorkspaceMode)
    if workspaceMode == "" {
        workspaceMode = "default"
    }
    fmt.Printf("  Mode: %s\n", workspaceMode)

    fmt.Printf("\nModel: %s\n", cfg.Agents.Defaults.Model)

    fmt.Println("\nProviders:")
    providers := map[string]string{
        "OpenRouter": cfg.Providers.OpenRouter.APIKey,
        "Claude":     cfg.Providers.Claude.APIKey,
        "OpenAI":     cfg.Providers.OpenAI.APIKey,
        "DeepSeek":   cfg.Providers.DeepSeek.APIKey,
        "Gemini":     cfg.Providers.Gemini.APIKey,
        "Ollama":     cfg.Providers.Ollama.BaseURL,
    }

    for name, key := range providers {
        status := "Not configured"
        if key != "" {
            status = "Configured"
        }
        fmt.Printf("  %s: %s\n", name, status)
    }

    fmt.Println("\nChannels:")
    fmt.Printf("  Telegram: %v\n", cfg.Channels.Telegram.Enabled)

    return nil
}
