package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
	"github.com/MEKXH/golem/internal/metrics"
	"github.com/MEKXH/golem/internal/skills"
	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show Golem configuration status",
		RunE:  runStatus,
	}
	cmd.Flags().Bool("json", false, "Output machine-readable JSON")
	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	jsonOutput := false
	if cmd != nil {
		jsonOutput, _ = cmd.Flags().GetBool("json")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return fmt.Errorf("invalid workspace: %w", err)
	}
	runtimeSnapshot, runtimeErr := metrics.ReadRuntimeSnapshot(workspacePath)
	if jsonOutput {
		return printStatusJSON(cfg, workspacePath, runtimeSnapshot, runtimeErr)
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

	// Runtime metrics
	fmt.Println("\nRuntime Metrics:")
	if runtimeErr != nil {
		fmt.Printf("  Status: unavailable (%v)\n", runtimeErr)
	} else if !runtimeSnapshot.HasData() {
		fmt.Println("  Status: no runtime data yet")
	} else {
		fmt.Printf("  updated_at=%s\n", runtimeSnapshot.UpdatedAt.Format(time.RFC3339))
		fmt.Printf(
			"  tool_total=%d tool_error_ratio=%.3f tool_timeout_ratio=%.3f tool_p95_proxy_ms=%d tool_avg_ms=%.1f\n",
			runtimeSnapshot.Tool.Total,
			runtimeSnapshot.Tool.ErrorRatio(),
			runtimeSnapshot.Tool.TimeoutRatio(),
			runtimeSnapshot.Tool.P95ProxyLatencyMs,
			runtimeSnapshot.Tool.AvgLatencyMs(),
		)
		fmt.Printf(
			"  channel_send_attempts=%d channel_send_failures=%d channel_send_failure_ratio=%.3f\n",
			runtimeSnapshot.Channel.SendAttempts,
			runtimeSnapshot.Channel.SendFailures,
			runtimeSnapshot.Channel.FailureRatio(),
		)
	}

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
	fmt.Println("  workflow: ready")

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

func printStatusJSON(cfg *config.Config, workspacePath string, runtimeSnapshot metrics.RuntimeSnapshot, runtimeErr error) error {
	configExists := false
	if _, err := os.Stat(config.ConfigPath()); err == nil {
		configExists = true
	}
	workspaceExists := false
	if _, err := os.Stat(workspacePath); err == nil {
		workspaceExists = true
	}
	workspaceMode := strings.TrimSpace(cfg.Agents.Defaults.WorkspaceMode)
	if workspaceMode == "" {
		workspaceMode = "default"
	}

	providers := map[string]bool{
		"openrouter": strings.TrimSpace(cfg.Providers.OpenRouter.APIKey) != "",
		"claude":     strings.TrimSpace(cfg.Providers.Claude.APIKey) != "",
		"openai":     strings.TrimSpace(cfg.Providers.OpenAI.APIKey) != "",
		"deepseek":   strings.TrimSpace(cfg.Providers.DeepSeek.APIKey) != "",
		"gemini":     strings.TrimSpace(cfg.Providers.Gemini.APIKey) != "",
		"ollama":     strings.TrimSpace(cfg.Providers.Ollama.BaseURL) != "",
	}

	toolsState := map[string]string{
		"read_file":           "ready",
		"write_file":          "ready",
		"edit_file":           "ready",
		"append_file":         "ready",
		"list_dir":            "ready",
		"read_memory":         "ready",
		"write_memory":        "ready",
		"append_diary":        "ready",
		"web_fetch":           "ready",
		"manage_cron":         "ready",
		"workflow":            "ready",
		"voice_transcription": "disabled",
	}
	if cfg.Tools.Voice.Enabled {
		toolsState["voice_transcription"] = fmt.Sprintf(
			"enabled (provider=%s, model=%s, timeout=%ds)",
			cfg.Tools.Voice.Provider,
			cfg.Tools.Voice.Model,
			cfg.Tools.Voice.TimeoutSeconds,
		)
	}
	toolsState["exec"] = fmt.Sprintf(
		"ready (timeout=%ds, restrict_to_workspace=%v)",
		cfg.Tools.Exec.Timeout,
		cfg.Tools.Exec.RestrictToWorkspace,
	)
	toolsState["web_search"] = "enabled (DuckDuckGo fallback)"
	if strings.TrimSpace(cfg.Tools.Web.Search.APIKey) != "" {
		toolsState["web_search"] = "enabled (Brave + DuckDuckGo fallback)"
	}

	cronStorePath := filepath.Join(workspacePath, "cron", "jobs.json")
	cronSvc := cron.NewService(cronStorePath, nil)
	cronTotal := 0
	cronEnabled := 0
	cronStatus := "ok"
	if err := cronSvc.Start(); err == nil {
		jobs := cronSvc.ListJobs(true)
		cronTotal = len(jobs)
		for _, j := range jobs {
			if j.Enabled {
				cronEnabled++
			}
		}
		cronSvc.Stop()
	} else {
		cronStatus = "unavailable"
	}

	loader := skills.NewLoader(workspacePath)
	skillList := loader.ListSkills()
	skillNames := make([]string, 0, len(skillList))
	for _, s := range skillList {
		skillNames = append(skillNames, s.Name)
	}

	payload := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"config": map[string]any{
			"path":   config.ConfigPath(),
			"exists": configExists,
		},
		"workspace": map[string]any{
			"path":   workspacePath,
			"exists": workspaceExists,
			"mode":   workspaceMode,
		},
		"model":           cfg.Agents.Defaults.Model,
		"runtime_metrics": runtimeSnapshot,
		"providers":       providers,
		"tools":           toolsState,
		"channels":        channelStates(cfg),
		"gateway": map[string]any{
			"host":             cfg.Gateway.Host,
			"port":             cfg.Gateway.Port,
			"token_configured": strings.TrimSpace(cfg.Gateway.Token) != "",
		},
		"cron": map[string]any{
			"status":  cronStatus,
			"total":   cronTotal,
			"enabled": cronEnabled,
		},
		"skills": map[string]any{
			"installed": len(skillList),
			"names":     skillNames,
		},
	}
	if runtimeErr != nil {
		payload["runtime_metrics_error"] = runtimeErr.Error()
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}
