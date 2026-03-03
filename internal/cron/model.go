package cron

import (
	"time"

	"github.com/google/uuid"
)

// Schedule 定义任务何时运行。
type Schedule struct {
	Kind    string `json:"kind"`               // "at" | "every" | "cron"
	AtMS    *int64 `json:"at_ms,omitempty"`    // one-shot timestamp (milliseconds)
	EveryMS *int64 `json:"every_ms,omitempty"` // interval (milliseconds)
	Expr    string `json:"expr,omitempty"`     // cron expression (5-field)
}

// Payload 定义任务触发时执行的操作。
type Payload struct {
	Kind    string `json:"kind"`    // "agent_turn"
	Message string `json:"message"` // message sent to agent
	Channel string `json:"channel"` // target channel name
	ChatID  string `json:"chat_id"` // target chat ID
	Deliver bool   `json:"deliver"` // true=deliver directly, false=agent processes
}

// JobState 保存任务的运行时状态。
type JobState struct {
	NextRunAtMS *int64 `json:"next_run_at_ms,omitempty"`
	LastRunAtMS *int64 `json:"last_run_at_ms,omitempty"`
	LastStatus  string `json:"last_status,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

// Job 表示一个定时任务。
type Job struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	Schedule       Schedule `json:"schedule"`
	Payload        Payload  `json:"payload"`
	State          JobState `json:"state"`
	CreatedAtMS    int64    `json:"created_at_ms"`
	UpdatedAtMS    int64    `json:"updated_at_ms"`
	DeleteAfterRun bool     `json:"delete_after_run,omitempty"`
}

// NewJob 创建一个带有生成 ID 和时间戳的新任务。
func NewJob(name string, schedule Schedule, payload Payload) *Job {
	now := time.Now().UnixMilli()
	return &Job{
		ID:          uuid.NewString()[:8],
		Name:        name,
		Enabled:     true,
		Schedule:    schedule,
		Payload:     payload,
		CreatedAtMS: now,
		UpdatedAtMS: now,
	}
}

// ShortID 返回用于显示的截断 ID。
func (j *Job) ShortID() string {
	if len(j.ID) > 8 {
		return j.ID[:8]
	}
	return j.ID
}

// ScheduleDescription 返回人类可读的调度摘要。
func (j *Job) ScheduleDescription() string {
	switch j.Schedule.Kind {
	case "at":
		if j.Schedule.AtMS != nil {
			t := time.UnixMilli(*j.Schedule.AtMS)
			return "at " + t.Format(time.RFC3339)
		}
		return "at (unset)"
	case "every":
		if j.Schedule.EveryMS != nil {
			d := time.Duration(*j.Schedule.EveryMS) * time.Millisecond
			return "every " + d.String()
		}
		return "every (unset)"
	case "cron":
		return "cron: " + j.Schedule.Expr
	default:
		return "unknown"
	}
}
