package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
	"github.com/MEKXH/golem/internal/metrics"
	"github.com/MEKXH/golem/internal/skills"
	"github.com/charmbracelet/lipgloss"
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

	// Styles
	var (
		headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#8E4EC6")).Padding(0, 1).MarginBottom(1)
		sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8E4EC6")).MarginTop(1).MarginBottom(0) // Purple
		keyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
		valStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
		okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#2E8B57")) // SeaGreen
		warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")) // Orange
		errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")) // OrangeRed
		dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	)

	fmt.Println(headerStyle.Render("Golem Status"))

	// Helper to print status line
	printStatus := func(label string, status string, isOk bool) {
		s := errorStyle
		if isOk {
			s = okStyle
		}
		fmt.Printf("  %s: %s\n", keyStyle.Render(label), s.Render(status))
	}

	fmt.Println(sectionStyle.Render("Config"))
	fmt.Printf("  %s: %s\n", keyStyle.Render("Path"), valStyle.Render(config.ConfigPath()))
	if _, err := os.Stat(config.ConfigPath()); err == nil {
		printStatus("Status", "OK", true)
	} else {
		printStatus("Status", "Not found (run 'golem init')", false)
	}

	fmt.Println(sectionStyle.Render("Workspace"))
	fmt.Printf("  %s: %s\n", keyStyle.Render("Path"), valStyle.Render(workspacePath))
	if _, err := os.Stat(workspacePath); err == nil {
		printStatus("Status", "OK", true)
	} else {
		printStatus("Status", "Not found", false)
	}
	workspaceMode := strings.TrimSpace(cfg.Agents.Defaults.WorkspaceMode)
	if workspaceMode == "" {
		workspaceMode = "default"
	}
	fmt.Printf("  %s: %s\n", keyStyle.Render("Mode"), valStyle.Render(workspaceMode))

	fmt.Println(sectionStyle.Render("Model"))
	fmt.Printf("  %s: %s\n", keyStyle.Render("Name"), valStyle.Render(cfg.Agents.Defaults.Model))

	// Runtime metrics
	fmt.Println(sectionStyle.Render("Runtime Metrics"))
	if runtimeErr != nil {
		fmt.Printf("  %s: %s\n", keyStyle.Render("Status"), errorStyle.Render(fmt.Sprintf("unavailable (%v)", runtimeErr)))
	} else if !runtimeSnapshot.HasData() {
		fmt.Printf("  %s: %s\n", keyStyle.Render("Status"), dimStyle.Render("no runtime data yet"))
	} else {
		fmt.Printf("  %s: %s\n", keyStyle.Render("updated_at"), valStyle.Render(runtimeSnapshot.UpdatedAt.Format(time.RFC3339)))
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

	fmt.Println(sectionStyle.Render("Providers"))
	providers := map[string]string{
		"OpenRouter": cfg.Providers.OpenRouter.APIKey,
		"Claude":     cfg.Providers.Claude.APIKey,
		"OpenAI":     cfg.Providers.OpenAI.APIKey,
		"DeepSeek":   cfg.Providers.DeepSeek.APIKey,
		"Gemini":     cfg.Providers.Gemini.APIKey,
		"Ollama":     cfg.Providers.Ollama.BaseURL,
	}

	providerNames := make([]string, 0, len(providers))
	for name := range providers {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)

	for _, name := range providerNames {
		key := providers[name]
		status := dimStyle.Render("Not configured")
		if key != "" {
			status = okStyle.Render("Configured")
		}
		fmt.Printf("  %s: %s\n", keyStyle.Render(name), status)
	}

	// Tools
	fmt.Println(sectionStyle.Render("Tools"))
	tools := []string{
		"read_file", "write_file", "edit_file", "append_file",
		"list_dir", "read_memory", "write_memory", "append_diary",
		"web_fetch", "manage_cron", "workflow",
	}
	for _, t := range tools {
		fmt.Printf("  %s: %s\n", keyStyle.Render(t), okStyle.Render("ready"))
	}
	fmt.Printf("  %s: %s\n", keyStyle.Render("exec"), okStyle.Render(fmt.Sprintf("ready (timeout=%ds, restrict_to_workspace=%v)", cfg.Tools.Exec.Timeout, cfg.Tools.Exec.RestrictToWorkspace)))

	webSearchStatus := okStyle.Render("enabled (DuckDuckGo fallback)")
	if strings.TrimSpace(cfg.Tools.Web.Search.APIKey) != "" {
		webSearchStatus = okStyle.Render("enabled (Brave + DuckDuckGo fallback)")
	}
	fmt.Printf("  %s: %s\n", keyStyle.Render("web_search"), webSearchStatus)

	voiceStatus := dimStyle.Render("disabled")
	if cfg.Tools.Voice.Enabled {
		voiceStatus = okStyle.Render(fmt.Sprintf(
			"enabled (provider=%s, model=%s, timeout=%ds)",
			cfg.Tools.Voice.Provider,
			cfg.Tools.Voice.Model,
			cfg.Tools.Voice.TimeoutSeconds,
		))
	}
	fmt.Printf("  %s: %s\n", keyStyle.Render("voice_transcription"), voiceStatus)

	// Channels
	fmt.Println(sectionStyle.Render("Channels"))
	for _, state := range channelStates(cfg) {
		line := dimStyle.Render("disabled")
		if state.Enabled {
			line = okStyle.Render("enabled")
			if state.Ready {
				line += okStyle.Render(" (ready)")
			} else {
				line += warnStyle.Render(" (" + state.Reason + ")")
			}
		}
		fmt.Printf("  %s: %s\n", keyStyle.Render(titleCase(state.Name)), line)
	}

	// Gateway
	fmt.Println(sectionStyle.Render("Gateway"))
	fmt.Printf("  %s: %s\n", keyStyle.Render("Address"), valStyle.Render(fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)))
	if cfg.Gateway.Token != "" {
		fmt.Printf("  %s: %s\n", keyStyle.Render("Auth"), okStyle.Render("token configured"))
	} else {
		fmt.Printf("  %s: %s\n", keyStyle.Render("Auth"), warnStyle.Render("no token (open)"))
	}

	// Cron
	fmt.Println(sectionStyle.Render("Cron"))
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
		fmt.Printf("  %s: %s\n", keyStyle.Render("Jobs"), valStyle.Render(fmt.Sprintf("%d total, %d enabled", len(jobs), enabled)))
		cronSvc.Stop()
	} else {
		fmt.Printf("  %s: %s\n", keyStyle.Render("Status"), errorStyle.Render("unavailable"))
	}

	// Skills
	fmt.Println(sectionStyle.Render("Skills"))
	loader := skills.NewLoader(workspacePath)
	skillList := loader.ListSkills()
	fmt.Printf("  %s: %s\n", keyStyle.Render("Installed"), valStyle.Render(fmt.Sprintf("%d", len(skillList))))
	for _, s := range skillList {
		fmt.Printf("    - %s (%s)\n", s.Name, dimStyle.Render(s.Source))
	}

	// Footer spacing
	fmt.Println()

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
