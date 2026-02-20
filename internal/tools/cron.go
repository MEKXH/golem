package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/cron"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type CronToolInput struct {
	Action       string `json:"action" jsonschema:"required,description=Action to perform: add list remove enable disable,enum=add,enum=list,enum=remove,enum=enable,enum=disable"`
	Name         string `json:"name,omitempty" jsonschema:"description=Job name (required for add)"`
	Message      string `json:"message,omitempty" jsonschema:"description=Message to send to agent (required for add)"`
	EverySeconds int64  `json:"every_seconds,omitempty" jsonschema:"description=Repeat interval in seconds (for add with every schedule)"`
	CronExpr     string `json:"cron_expr,omitempty" jsonschema:"description=Cron expression like '0 9 * * *' (for add with cron schedule)"`
	AtTimestamp  string `json:"at_timestamp,omitempty" jsonschema:"description=RFC3339 timestamp for one-shot (for add with at schedule)"`
	JobID        string `json:"job_id,omitempty" jsonschema:"description=Job ID (required for remove/enable/disable)"`
	Deliver      bool   `json:"deliver,omitempty" jsonschema:"description=If true deliver response directly without agent processing"`
}

type cronToolImpl struct {
	service *cron.Service
}

func (t *cronToolImpl) execute(ctx context.Context, input *CronToolInput) (string, error) {
	switch strings.ToLower(strings.TrimSpace(input.Action)) {
	case "add":
		return t.add(input)
	case "list":
		return t.list()
	case "remove":
		return t.remove(input)
	case "enable":
		return t.enable(input, true)
	case "disable":
		return t.enable(input, false)
	default:
		return "", fmt.Errorf("unknown action: %s (expected: add, list, remove, enable, disable)", input.Action)
	}
}

func (t *cronToolImpl) add(input *CronToolInput) (string, error) {
	if strings.TrimSpace(input.Name) == "" {
		return "", fmt.Errorf("name is required for add action")
	}
	if strings.TrimSpace(input.Message) == "" {
		return "", fmt.Errorf("message is required for add action")
	}

	var schedule cron.Schedule
	switch {
	case input.EverySeconds > 0:
		ms := input.EverySeconds * 1000
		schedule = cron.Schedule{Kind: "every", EveryMS: &ms}
	case strings.TrimSpace(input.CronExpr) != "":
		schedule = cron.Schedule{Kind: "cron", Expr: strings.TrimSpace(input.CronExpr)}
	case strings.TrimSpace(input.AtTimestamp) != "":
		ts, err := time.Parse(time.RFC3339, strings.TrimSpace(input.AtTimestamp))
		if err != nil {
			return "", fmt.Errorf("invalid at_timestamp (expected RFC3339): %w", err)
		}
		ms := ts.UnixMilli()
		schedule = cron.Schedule{Kind: "at", AtMS: &ms}
	default:
		return "", fmt.Errorf("one of every_seconds, cron_expr, or at_timestamp is required")
	}

	job, err := t.service.AddJob(input.Name, input.Message, schedule, "", "", input.Deliver)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Job created: id=%s name=%s schedule=%s", job.ShortID(), job.Name, job.ScheduleDescription()), nil
}

func (t *cronToolImpl) list() (string, error) {
	jobs := t.service.ListJobs(true)
	if len(jobs) == 0 {
		return "No scheduled jobs.", nil
	}

	type jobView struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Enabled  bool   `json:"enabled"`
		Schedule string `json:"schedule"`
		NextRun  string `json:"next_run,omitempty"`
	}

	views := make([]jobView, 0, len(jobs))
	for _, j := range jobs {
		v := jobView{
			ID:       j.ShortID(),
			Name:     j.Name,
			Enabled:  j.Enabled,
			Schedule: j.ScheduleDescription(),
		}
		if j.State.NextRunAtMS != nil {
			v.NextRun = time.UnixMilli(*j.State.NextRunAtMS).Format(time.RFC3339)
		}
		views = append(views, v)
	}

	data, _ := json.MarshalIndent(views, "", "  ")
	return string(data), nil
}

func (t *cronToolImpl) remove(input *CronToolInput) (string, error) {
	if strings.TrimSpace(input.JobID) == "" {
		return "", fmt.Errorf("job_id is required for remove action")
	}
	if err := t.service.RemoveJob(input.JobID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Job %s removed.", input.JobID), nil
}

func (t *cronToolImpl) enable(input *CronToolInput, enabled bool) (string, error) {
	if strings.TrimSpace(input.JobID) == "" {
		return "", fmt.Errorf("job_id is required for enable/disable action")
	}
	job, err := t.service.EnableJob(input.JobID, enabled)
	if err != nil {
		return "", err
	}
	state := "enabled"
	if !enabled {
		state = "disabled"
	}
	return fmt.Sprintf("Job %s (%s) %s.", job.ShortID(), job.Name, state), nil
}

// NewCronTool creates an agent tool for managing cron jobs.
func NewCronTool(service *cron.Service) (tool.InvokableTool, error) {
	impl := &cronToolImpl{service: service}
	return utils.InferTool(
		"manage_cron",
		"Create, list, remove, enable, or disable scheduled (cron) jobs. Jobs send messages to the agent on a schedule.",
		impl.execute,
	)
}
