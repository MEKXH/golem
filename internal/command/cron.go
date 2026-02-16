package command

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/cron"
)

// CronCommand implements /cron — manage scheduled jobs.
// Subcommands: list, remove <id>, enable <id>, disable <id>, run <id>
type CronCommand struct{}

func (c *CronCommand) Name() string        { return "cron" }
func (c *CronCommand) Description() string { return "Manage cron jobs (list|remove|enable|disable|run)" }

func (c *CronCommand) Execute(ctx context.Context, args string, env Env) Result {
	sub, rest, _ := strings.Cut(args, " ")
	sub = strings.ToLower(strings.TrimSpace(sub))
	rest = strings.TrimSpace(rest)

	storePath := filepath.Join(env.WorkspacePath, "cron", "jobs.json")
	svc := cron.NewService(storePath, nil)
	if err := svc.Start(); err != nil {
		return Result{Content: fmt.Sprintf("Error loading cron: %v", err)}
	}
	defer svc.Stop()

	switch sub {
	case "", "list":
		return cronList(svc)
	case "remove", "rm", "delete":
		return cronRemove(svc, rest)
	case "enable":
		return cronSetEnabled(svc, rest, true)
	case "disable":
		return cronSetEnabled(svc, rest, false)
	case "run":
		return cronRun(svc, rest)
	default:
		return Result{Content: "Usage: `/cron [list|remove|enable|disable|run] [id]`"}
	}
}

func cronList(svc *cron.Service) Result {
	jobs := svc.ListJobs(true)
	if len(jobs) == 0 {
		return Result{Content: "No cron jobs."}
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Cron jobs (%d):**\n\n", len(jobs)))
	for _, j := range jobs {
		status := "enabled"
		if !j.Enabled {
			status = "disabled"
		}
		next := "-"
		if j.State.NextRunAtMS != nil {
			next = time.UnixMilli(*j.State.NextRunAtMS).Format("01-02 15:04")
		}
		sb.WriteString(fmt.Sprintf("- `%s` **%s** — %s (%s) next: %s\n",
			shortID(j.ID), j.Name, j.ScheduleDescription(), status, next))
	}
	return Result{Content: sb.String()}
}

func cronRemove(svc *cron.Service, id string) Result {
	if id == "" {
		return Result{Content: "Usage: `/cron remove <id>`"}
	}
	if err := svc.RemoveJob(id); err != nil {
		return Result{Content: fmt.Sprintf("Error: %v", err)}
	}
	return Result{Content: fmt.Sprintf("Job `%s` removed.", shortID(id))}
}

func cronSetEnabled(svc *cron.Service, id string, enabled bool) Result {
	if id == "" {
		action := "enable"
		if !enabled {
			action = "disable"
		}
		return Result{Content: fmt.Sprintf("Usage: `/cron %s <id>`", action)}
	}
	job, err := svc.EnableJob(id, enabled)
	if err != nil {
		return Result{Content: fmt.Sprintf("Error: %v", err)}
	}
	state := "enabled"
	if !job.Enabled {
		state = "disabled"
	}
	return Result{Content: fmt.Sprintf("Job `%s` (**%s**) is now %s.", shortID(job.ID), job.Name, state)}
}

func cronRun(svc *cron.Service, id string) Result {
	if id == "" {
		return Result{Content: "Usage: `/cron run <id>`"}
	}
	job, err := svc.RunJob(id)
	if err != nil {
		return Result{Content: fmt.Sprintf("Error: %v", err)}
	}
	if job == nil {
		return Result{Content: fmt.Sprintf("Job `%s` executed and removed (one-shot).", shortID(id))}
	}
	lastStatus := job.State.LastStatus
	if lastStatus == "" {
		lastStatus = "ok"
	}
	return Result{Content: fmt.Sprintf("Job `%s` (**%s**) executed: %s", shortID(job.ID), job.Name, lastStatus)}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
