package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/agent"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
	"github.com/MEKXH/golem/internal/provider"
	"github.com/spf13/cobra"
)

func NewCronCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Manage scheduled tasks",
	}

	cmd.AddCommand(
		newCronListCmd(),
		newCronAddCmd(),
		newCronRunCmd(),
		newCronRemoveCmd(),
		newCronEnableCmd(),
		newCronDisableCmd(),
	)

	return cmd
}

func newCronListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all scheduled jobs",
		RunE:  runCronList,
	}
}

func newCronAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new scheduled job",
		RunE:  runCronAdd,
	}

	cmd.Flags().StringP("name", "n", "", "Job name (required)")
	cmd.Flags().StringP("message", "m", "", "Message to send to agent (required)")
	cmd.Flags().Int64("every", 0, "Repeat interval in seconds")
	cmd.Flags().String("cron", "", "Cron expression (e.g., '0 9 * * *')")
	cmd.Flags().String("at", "", "One-shot timestamp (RFC3339)")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("message")

	return cmd
}

func newCronRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <job_id>",
		Short: "Run a scheduled job immediately",
		Args:  cobra.ExactArgs(1),
		RunE:  runCronNow,
	}
}

func newCronRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <job_id>",
		Short: "Remove a scheduled job",
		Args:  cobra.ExactArgs(1),
		RunE:  runCronRemove,
	}
}

func newCronEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <job_id>",
		Short: "Enable a scheduled job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCronSetEnabled(args[0], true)
		},
	}
}

func newCronDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <job_id>",
		Short: "Disable a scheduled job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCronSetEnabled(args[0], false)
		},
	}
}

func loadCronService() (*cron.Service, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return nil, fmt.Errorf("invalid workspace: %w", err)
	}
	storePath := filepath.Join(workspacePath, "cron", "jobs.json")
	svc := cron.NewService(storePath, nil)
	if err := svc.Start(); err != nil {
		return nil, err
	}
	return svc, nil
}

func runCronList(cmd *cobra.Command, args []string) error {
	svc, err := loadCronService()
	if err != nil {
		return err
	}
	defer svc.Stop()

	jobs := svc.ListJobs(true)
	if len(jobs) == 0 {
		fmt.Println("No scheduled jobs.")
		return nil
	}

	fmt.Printf("  %-10s %-20s %-25s %-22s %s\n", "ID", "NAME", "SCHEDULE", "NEXT RUN", "STATUS")
	fmt.Printf("  %-10s %-20s %-25s %-22s %s\n",
		strings.Repeat("-", 10),
		strings.Repeat("-", 20),
		strings.Repeat("-", 25),
		strings.Repeat("-", 22),
		strings.Repeat("-", 10),
	)

	for _, j := range jobs {
		nextRun := "-"
		if j.State.NextRunAtMS != nil {
			nextRun = time.UnixMilli(*j.State.NextRunAtMS).Format("2006-01-02 15:04:05")
		}
		status := "enabled"
		if !j.Enabled {
			status = "disabled"
		}
		fmt.Printf("  %-10s %-20s %-25s %-22s %s\n",
			j.ShortID(),
			truncate(j.Name, 20),
			truncate(j.ScheduleDescription(), 25),
			nextRun,
			status,
		)
	}

	return nil
}

func runCronAdd(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	message, _ := cmd.Flags().GetString("message")
	every, _ := cmd.Flags().GetInt64("every")
	cronExpr, _ := cmd.Flags().GetString("cron")
	at, _ := cmd.Flags().GetString("at")

	var schedule cron.Schedule
	switch {
	case every > 0:
		ms := every * 1000
		schedule = cron.Schedule{Kind: "every", EveryMS: &ms}
	case cronExpr != "":
		schedule = cron.Schedule{Kind: "cron", Expr: cronExpr}
	case at != "":
		ts, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return fmt.Errorf("invalid --at timestamp (expected RFC3339): %w", err)
		}
		ms := ts.UnixMilli()
		schedule = cron.Schedule{Kind: "at", AtMS: &ms}
	default:
		return fmt.Errorf("one of --every, --cron, or --at is required")
	}

	svc, err := loadCronService()
	if err != nil {
		return err
	}
	defer svc.Stop()

	job, err := svc.AddJob(name, message, schedule, "", "", false)
	if err != nil {
		return err
	}

	fmt.Printf("Job created: %s (%s)\n", job.ShortID(), job.ScheduleDescription())
	return nil
}

func runCronRemove(cmd *cobra.Command, args []string) error {
	svc, err := loadCronService()
	if err != nil {
		return err
	}
	defer svc.Stop()

	if err := svc.RemoveJob(args[0]); err != nil {
		return err
	}
	fmt.Printf("Job %s removed.\n", args[0])
	return nil
}

func runCronSetEnabled(jobID string, enabled bool) error {
	svc, err := loadCronService()
	if err != nil {
		return err
	}
	defer svc.Stop()

	job, err := svc.EnableJob(jobID, enabled)
	if err != nil {
		return err
	}
	state := "enabled"
	if !enabled {
		state = "disabled"
	}
	fmt.Printf("Job %s (%s) %s.\n", job.ShortID(), job.Name, state)
	return nil
}

func runCronNow(cmd *cobra.Command, args []string) error {
	jobID := strings.TrimSpace(args[0])
	if jobID == "" {
		return fmt.Errorf("job_id is required")
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return fmt.Errorf("invalid workspace: %w", err)
	}

	msgBus := bus.NewMessageBus(10)
	model, err := provider.NewChatModel(ctx, cfg)
	if err != nil {
		fmt.Printf("Warning: %v\nRunning without LLM (job may produce fallback response)\n", err)
		model = nil
	}

	loop, err := agent.NewLoop(cfg, msgBus, model)
	if err != nil {
		return fmt.Errorf("invalid workspace: %w", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	cronStorePath := filepath.Join(workspacePath, "cron", "jobs.json")
	svc := cron.NewService(cronStorePath, func(job *cron.Job) error {
		ch := strings.TrimSpace(job.Payload.Channel)
		if ch == "" {
			ch = "cron"
		}
		chatID := strings.TrimSpace(job.Payload.ChatID)
		if chatID == "" {
			chatID = "default"
		}
		_, err := loop.ProcessForChannel(ctx, ch, chatID, "cron", job.Payload.Message)
		return err
	})
	if err := svc.Start(); err != nil {
		return err
	}
	defer svc.Stop()

	job, err := svc.RunJob(jobID)
	if err != nil {
		return err
	}

	if job == nil {
		fmt.Printf("Job %s executed (one-shot job removed after run).\n", jobID)
		return nil
	}

	status := job.State.LastStatus
	if status == "" {
		status = "unknown"
	}
	fmt.Printf("Job %s (%s) executed, status=%s.\n", job.ShortID(), job.Name, status)
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
