package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
	"github.com/MEKXH/golem/internal/skills"
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

	// Tools
	fmt.Println("\nTools:")
	fmt.Println("  read_file: ready")
	fmt.Println("  write_file: ready")
	fmt.Println("  edit_file: ready")
	fmt.Println("  append_file: ready")
	fmt.Println("  list_dir: ready")
	fmt.Println("  read_memory: ready")
	fmt.Println("  write_memory: ready")
	fmt.Println("  append_diary: ready")
	fmt.Printf("  exec: ready (timeout=%ds, restrict_to_workspace=%v)\n", cfg.Tools.Exec.Timeout, cfg.Tools.Exec.RestrictToWorkspace)
	fmt.Println("  web_fetch: ready")
	webSearchStatus := "enabled (DuckDuckGo fallback)"
	if strings.TrimSpace(cfg.Tools.Web.Search.APIKey) != "" {
		webSearchStatus = "enabled (Brave + DuckDuckGo fallback)"
	}
	fmt.Printf("  web_search: %s\n", webSearchStatus)
	voiceStatus := "disabled"
	if cfg.Tools.Voice.Enabled {
		voiceStatus = fmt.Sprintf(
			"enabled (provider=%s, model=%s, timeout=%ds)",
			cfg.Tools.Voice.Provider,
			cfg.Tools.Voice.Model,
			cfg.Tools.Voice.TimeoutSeconds,
		)
	}
	fmt.Printf("  voice_transcription: %s\n", voiceStatus)
	fmt.Println("  manage_cron: ready")

	// Channels
	fmt.Println("\nChannels:")
	for _, state := range channelStates(cfg) {
		line := "disabled"
		if state.Enabled {
			line = "enabled"
			if state.Ready {
				line += " (ready)"
			} else {
				line += " (" + state.Reason + ")"
			}
		}
		fmt.Printf("  %s: %s\n", titleCase(state.Name), line)
	}

	// Gateway
	fmt.Println("\nGateway:")
	fmt.Printf("  Address: %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)
	if cfg.Gateway.Token != "" {
		fmt.Println("  Auth:    token configured")
	} else {
		fmt.Println("  Auth:    no token (open)")
	}

	// Cron
	fmt.Println("\nCron:")
	cronStorePath := filepath.Join(workspacePath, "cron", "jobs.json")
	cronSvc := cron.NewService(cronStorePath, nil)
	if err := cronSvc.Start(); err == nil {
		jobs := cronSvc.ListJobs(true)
		enabled := 0
		for _, j := range jobs {
			if j.Enabled {
				enabled++
			}
		}
		fmt.Printf("  Jobs: %d total, %d enabled\n", len(jobs), enabled)
		cronSvc.Stop()
	} else {
		fmt.Println("  Status: unavailable")
	}

	// Skills
	fmt.Println("\nSkills:")
	loader := skills.NewLoader(workspacePath)
	skillList := loader.ListSkills()
	fmt.Printf("  Installed: %d\n", len(skillList))
	for _, s := range skillList {
		fmt.Printf("    - %s (%s)\n", s.Name, s.Source)
	}

	return nil
}
