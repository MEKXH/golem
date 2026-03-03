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

// StatusCommand 实现 /status 命令 — 用于显示 Agent 当前的运行时状态、配置概览及性能指标。
type StatusCommand struct{}

// Name 返回命令名称。
func (c *StatusCommand) Name() string        { return "status" }

// Description 返回命令描述。
func (c *StatusCommand) Description() string { return "Show runtime status" }

// Execute 执行显示状态摘要的逻辑。
func (c *StatusCommand) Execute(_ context.Context, _ string, env Env) Result {
	var sb strings.Builder
	sb.WriteString("**Golem Status**\n\n")

	// 1. 模型与工作区信息
	if env.Config != nil {
		sb.WriteString(fmt.Sprintf("- **Model:** `%s`\n", env.Config.Agents.Defaults.Model))
	}
	sb.WriteString(fmt.Sprintf("- **Workspace:** `%s`\n", env.WorkspacePath))

	// 2. LLM 供应商配置状态
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

	// 3. 运行时性能指标 (Metrics)
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

	// 4. Cron 定时任务统计
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

	// 5. 已安装技能统计
	loader := skills.NewLoader(env.WorkspacePath)
	skillList := loader.ListSkills()
	sb.WriteString(fmt.Sprintf("- **Skills:** %d installed\n", len(skillList)))

	// 6. 配置文件路径
	configStatus := ""
	if _, err := os.Stat(config.ConfigPath()); err != nil {
		configStatus = " (not found)"
	}
	sb.WriteString(fmt.Sprintf("- **Config:** `%s`%s\n", config.ConfigPath(), configStatus))

	return Result{Content: sb.String()}
}
