package cron

import (
	"time"

	"github.com/google/uuid"
)

// Schedule defines when a job should run.
type Schedule struct {
	Kind    string `json:"kind"`               // "at" | "every" | "cron"
	AtMS    *int64 `json:"at_ms,omitempty"`     // one-shot timestamp (milliseconds)
	EveryMS *int64 `json:"every_ms,omitempty"`  // interval (milliseconds)
	Expr    string `json:"expr,omitempty"`      // cron expression (5-field)
}

// Payload defines what a job does when triggered.
type Payload struct {
	Kind    string `json:"kind"`    // "agent_turn"
	Message string `json:"message"` // message sent to agent
	Channel string `json:"channel"` // target channel name
	ChatID  string `json:"chat_id"` // target chat ID
	Deliver bool   `json:"deliver"` // true=deliver directly, false=agent processes
}

// JobState holds runtime state for a job.
type JobState struct {
	NextRunAtMS *int64 `json:"next_run_at_ms,omitempty"`
	LastRunAtMS *int64 `json:"last_run_at_ms,omitempty"`
	LastStatus  string `json:"last_status,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

// Job represents a scheduled task.
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

// NewJob creates a new job with a generated ID and timestamps.
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

// ShortID returns a truncated ID for display.
func (j *Job) ShortID() string {
	if len(j.ID) > 8 {
		return j.ID[:8]
	}
	return j.ID
}

// ScheduleDescription returns a human-readable schedule summary.
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
