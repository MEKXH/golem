package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
	"github.com/MEKXH/golem/internal/metrics"
	"github.com/MEKXH/golem/internal/skills"
)

// StatusCommand implements /status — shows runtime status summary.
type StatusCommand struct{}

func (c *StatusCommand) Name() string        { return "status" }
func (c *StatusCommand) Description() string { return "Show runtime status" }

func (c *StatusCommand) Execute(_ context.Context, _ string, env Env) Result {
	var sb strings.Builder
	sb.WriteString("**Golem Status**\n\n")

	// Model & Workspace
	if env.Config != nil {
		sb.WriteString(fmt.Sprintf("- **Model:** `%s`\n", env.Config.Agents.Defaults.Model))
	}
	sb.WriteString(fmt.Sprintf("- **Workspace:** `%s`\n", env.WorkspacePath))

	// Providers
	if env.Config != nil {
		sb.WriteString("\n**Providers:**\n\n")
		for name, key := range map[string]string{
			"OpenRouter": env.Config.Providers.OpenRouter.APIKey,
			"Claude":     env.Config.Providers.Claude.APIKey,
			"OpenAI":     env.Config.Providers.OpenAI.APIKey,
			"DeepSeek":   env.Config.Providers.DeepSeek.APIKey,
			"Gemini":     env.Config.Providers.Gemini.APIKey,
			"Ollama":     env.Config.Providers.Ollama.BaseURL,
		} {
			status := "not configured"
			if strings.TrimSpace(key) != "" {
				status = "configured"
			}
			sb.WriteString(fmt.Sprintf("- %s: %s\n", name, status))
		}
	}

	// Runtime metrics
	sb.WriteString("\n**Metrics:**\n\n")
	if env.Metrics != nil {
		snap := env.Metrics.Snapshot()
		if !snap.HasData() {
			snap, _ = metrics.ReadRuntimeSnapshot(env.WorkspacePath)
		}
		if snap.HasData() {
			sb.WriteString(fmt.Sprintf("- Updated: `%s`\n", snap.UpdatedAt.Format(time.RFC3339)))
			sb.WriteString(fmt.Sprintf("- Tools: %d calls, err=%.1f%%, p95=%dms\n",
				snap.Tool.Total,
				snap.Tool.ErrorRatio()*100,
				snap.Tool.P95ProxyLatencyMs,
			))
			sb.WriteString(fmt.Sprintf("- Channel: %d sends, fail=%.1f%%\n",
				snap.Channel.SendAttempts,
				snap.Channel.FailureRatio()*100,
			))
		} else {
			sb.WriteString("- No data yet\n")
		}
	} else {
		sb.WriteString("- Unavailable\n")
	}

	// Cron
	cronStorePath := filepath.Join(env.WorkspacePath, "cron", "jobs.json")
	cronSvc := cron.NewService(cronStorePath, nil)
	if err := cronSvc.Start(); err == nil {
		jobs := cronSvc.ListJobs(true)
		enabled := 0
		for _, j := range jobs {
			if j.Enabled {
				enabled++
			}
		}
		sb.WriteString(fmt.Sprintf("\n- **Cron:** %d jobs (%d enabled)\n", len(jobs), enabled))
		cronSvc.Stop()
	}

	// Skills
	loader := skills.NewLoader(env.WorkspacePath)
	skillList := loader.ListSkills()
	sb.WriteString(fmt.Sprintf("- **Skills:** %d installed\n", len(skillList)))

	// Config path
	configStatus := ""
	if _, err := os.Stat(config.ConfigPath()); err != nil {
		configStatus = " (not found)"
	}
	sb.WriteString(fmt.Sprintf("- **Config:** `%s`%s\n", config.ConfigPath(), configStatus))

	return Result{Content: sb.String()}
}
