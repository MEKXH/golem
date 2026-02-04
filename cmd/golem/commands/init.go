package commands

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/MEKXH/golem/internal/config"
    "github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "init",
        Short: "Initialize Golem configuration",
        RunE:  runInit,
    }
}

func runInit(cmd *cobra.Command, args []string) error {
    configPath := config.ConfigPath()

    if _, err := os.Stat(configPath); err == nil {
        fmt.Printf("Config already exists: %s\n", configPath)
        return nil
    }

    cfg := config.DefaultConfig()
    workspacePath, err := cfg.WorkspacePathChecked()
    if err != nil {
        return fmt.Errorf("invalid workspace: %w", err)
    }

    dirs := []string{
        config.ConfigDir(),
        workspacePath,
        filepath.Join(workspacePath, "memory"),
        filepath.Join(workspacePath, "skills"),
        filepath.Join(config.ConfigDir(), "sessions"),
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir, err)
        }
    }

    if err := config.Save(cfg); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    workspaceFiles := map[string]string{
        "IDENTITY.md": "# Identity\n\nYou are Golem, a helpful AI assistant.",
        "SOUL.md":     "# Soul\n\nBe helpful, concise, and proactive.",
        "USER.md":     "# User\n\nInformation about the user goes here.",
        "AGENTS.md":   "# Agents\n\nAgent-specific instructions go here.",
    }

    for name, content := range workspaceFiles {
        path := filepath.Join(workspacePath, name)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            _ = os.WriteFile(path, []byte(content), 0644)
        }
    }

    fmt.Printf("Golem initialized!\n")
    fmt.Printf("Config: %s\n", configPath)
    fmt.Printf("Workspace: %s\n", workspacePath)
    fmt.Printf("\nNext steps:\n")
    fmt.Printf("1. Edit %s to add your API keys\n", configPath)
    fmt.Printf("2. Run 'golem chat' to start chatting\n")

    return nil
}
