package cron

import (
	"time"

	"github.com/google/uuid"
)

// Schedule 定义任务触发的计划规则。
type Schedule struct {
	Kind    string `json:"kind"`               // 调度类型："at" (单次), "every" (间隔), "cron" (表达式)
	AtMS    *int64 `json:"at_ms,omitempty"`    // 单次执行的时间戳（毫秒）
	EveryMS *int64 `json:"every_ms,omitempty"` // 执行间隔时长（毫秒）
	Expr    string `json:"expr,omitempty"`     // 5 段式 Cron 表达式
}

// Payload 描述定时任务触发时需要执行的具体操作。
type Payload struct {
	Kind    string `json:"kind"`    // 载荷类型（如 "agent_turn"）
	Message string `json:"message"` // 发送给 Agent 的指令内容
	Channel string `json:"channel"` // 目标回复通道名称
	ChatID  string `json:"chat_id"` // 目标聊天 ID
	Deliver bool   `json:"deliver"` // 是否直接交付结果而跳过 Agent 思考
}

// JobState 维护任务的实时运行状态。
type JobState struct {
	NextRunAtMS *int64 `json:"next_run_at_ms,omitempty"` // 下次预定执行时间
	LastRunAtMS *int64 `json:"last_run_at_ms,omitempty"` // 最近一次执行时间
	LastStatus  string `json:"last_status,omitempty"`    // 最近执行状态 ("ok", "error")
	LastError   string `json:"last_error,omitempty"`     // 最近一次执行产生的错误消息
}

// Job 表示一个完整的定时任务配置及其状态。
type Job struct {
	ID             string   `json:"id"`               // 唯一标识符
	Name           string   `json:"name"`             // 任务显示名称
	Enabled        bool     `json:"enabled"`          // 是否处于启用状态
	Schedule       Schedule `json:"schedule"`         // 调度计划
	Payload        Payload  `json:"payload"`          // 待执行载荷
	State          JobState `json:"state"`            // 运行状态
	CreatedAtMS    int64    `json:"created_at_ms"`    // 创建时间
	UpdatedAtMS    int64    `json:"updated_at_ms"`    // 更新时间
	DeleteAfterRun bool     `json:"delete_after_run,omitempty"` // 执行完成后是否自动从存储中删除
}

// NewJob 创建一个新的任务实例，并自动分配 ID 和时间戳。
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

// ShortID 返回任务的简短 ID（前 8 位）。
func (j *Job) ShortID() string {
	if len(j.ID) > 8 {
		return j.ID[:8]
	}
	return j.ID
}

// ScheduleDescription 返回任务调度计划的人类可读描述。
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
