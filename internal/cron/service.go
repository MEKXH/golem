// Package cron 实现 Golem 的定时任务 (Cron Jobs) 调度系统。
package cron

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/adhocore/gronx"
)

// JobHandler 定义了定时任务触发时的处理函数。
type JobHandler func(*Job) error

// Service 负责管理定时任务的生命周期，并使用轮询循环 (Ticker) 进行调度执行。
type Service struct {
	store    *Store        // 任务持久化存储
	onJob    JobHandler    // 任务触发时的回调
	mu       sync.RWMutex
	stopChan chan struct{}
	stopped  chan struct{}
	running  bool
}

// NewService 创建并返回一个由指定文件路径支持的定时任务服务。
func NewService(storePath string, handler JobHandler) *Service {
	return &Service{
		store: NewStore(storePath),
		onJob: handler,
	}
}

// Start 从磁盘加载任务并启动调度轮询循环。
func (s *Service) Start() error {
	if err := s.store.Load(); err != nil {
		return fmt.Errorf("cron service start: %w", err)
	}

	// 为所有已启用但未计算下次运行时间的任务初始化运行时间
	for _, job := range s.store.All() {
		if job.Enabled && job.State.NextRunAtMS == nil {
			s.computeNextRun(job)
			s.store.Put(job)
		}
	}
	if err := s.store.Save(); err != nil {
		slog.Warn("cron: failed to save after init", "error", err)
	}

	s.mu.Lock()
	s.stopChan = make(chan struct{})
	s.stopped = make(chan struct{})
	s.running = true
	s.mu.Unlock()

	go s.loop()

	slog.Info("cron service started", "jobs", len(s.store.All()))
	return nil
}

// Stop 停止调度循环并优雅退出。
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	<-s.stopped
	slog.Info("cron service stopped")
}

func (s *Service) loop() {
	defer close(s.stopped)

	// 每秒检查一次是否有任务到期
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Service) tick() {
	now := time.Now().UnixMilli()
	jobs := s.store.All()

	var due []*Job
	for _, j := range jobs {
		if !j.Enabled {
			continue
		}
		if j.State.NextRunAtMS == nil {
			continue
		}
		// 检查任务是否已到期
		if *j.State.NextRunAtMS <= now {
			// 清除下次运行时间以防重复触发（随后会在执行后重新计算）
			j.State.NextRunAtMS = nil
			s.store.Put(j)
			due = append(due, j)
		}
	}

	// 执行到期的任务
	for _, j := range due {
		s.executeJob(j)
	}
}

func (s *Service) executeJob(job *Job) {
	slog.Info("cron: executing job", "id", job.ID, "name", job.Name)

	now := time.Now().UnixMilli()
	var execErr error
	if s.onJob != nil {
		execErr = s.onJob(job)
	}

	job.State.LastRunAtMS = &now
	if execErr != nil {
		job.State.LastStatus = "error"
		job.State.LastError = execErr.Error()
		slog.Error("cron: job execution failed", "id", job.ID, "error", execErr)
	} else {
		job.State.LastStatus = "ok"
		job.State.LastError = ""
	}

	job.UpdatedAtMS = now

	// 根据调度类型处理后续状态
	switch job.Schedule.Kind {
	case "at":
		// 单次任务执行后删除或禁用
		if job.DeleteAfterRun {
			s.store.Delete(job.ID)
		} else {
			job.Enabled = false
			s.store.Put(job)
		}
	default:
		// 周期性任务计算下次运行时间
		s.computeNextRun(job)
		s.store.Put(job)
	}

	if err := s.store.Save(); err != nil {
		slog.Warn("cron: failed to save after execution", "error", err)
	}
}

// RunJob 立即运行指定的任务（忽略其预定的运行时间）。
func (s *Service) RunJob(id string) (*Job, error) {
	job, ok := s.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	s.executeJob(job)

	updated, ok := s.store.Get(id)
	if ok {
		return updated, nil
	}
	return nil, nil
}

func (s *Service) computeNextRun(job *Job) {
	now := time.Now()

	switch job.Schedule.Kind {
	case "at":
		if job.Schedule.AtMS != nil {
			ms := *job.Schedule.AtMS
			job.State.NextRunAtMS = &ms
		}
	case "every":
		if job.Schedule.EveryMS != nil {
			next := now.Add(time.Duration(*job.Schedule.EveryMS) * time.Millisecond).UnixMilli()
			job.State.NextRunAtMS = &next
		}
	case "cron":
		if job.Schedule.Expr != "" {
			nextTime, err := gronx.NextTickAfter(job.Schedule.Expr, now, false)
			if err != nil {
				slog.Warn("cron: failed to compute next run", "id", job.ID, "expr", job.Schedule.Expr, "error", err)
				return
			}
			ms := nextTime.UnixMilli()
			job.State.NextRunAtMS = &ms
		}
	}
}

// AddJob 创建并持久化一个新的定时任务。
func (s *Service) AddJob(name, message string, schedule Schedule, channel, chatID string, deliver bool) (*Job, error) {
	payload := Payload{
		Kind:    "agent_turn",
		Message: message,
		Channel: channel,
		ChatID:  chatID,
		Deliver: deliver,
	}

	job := NewJob(name, schedule, payload)

	if schedule.Kind == "at" {
		job.DeleteAfterRun = true
	}

	s.computeNextRun(job)
	s.store.Put(job)

	if err := s.store.Save(); err != nil {
		return nil, fmt.Errorf("save job: %w", err)
	}

	slog.Info("cron: job added", "id", job.ID, "name", name, "schedule", job.ScheduleDescription())
	return job, nil
}

// RemoveJob 按 ID 删除指定的任务。
func (s *Service) RemoveJob(id string) error {
	if !s.store.Delete(id) {
		return fmt.Errorf("job not found: %s", id)
	}
	if err := s.store.Save(); err != nil {
		return fmt.Errorf("save after remove: %w", err)
	}
	return nil
}

// EnableJob 启用或禁用指定的任务。
func (s *Service) EnableJob(id string, enabled bool) (*Job, error) {
	job, ok := s.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	job.Enabled = enabled
	job.UpdatedAtMS = time.Now().UnixMilli()

	if enabled && job.State.NextRunAtMS == nil {
		s.computeNextRun(job)
	}

	s.store.Put(job)
	if err := s.store.Save(); err != nil {
		return nil, fmt.Errorf("save after enable: %w", err)
	}
	return job, nil
}

// ListJobs 返回所有任务列表，可按创建时间排序。
func (s *Service) ListJobs(includeDisabled bool) []*Job {
	all := s.store.All()
	if includeDisabled {
		sort.Slice(all, func(i, j int) bool { return all[i].CreatedAtMS < all[j].CreatedAtMS })
		return all
	}
	var result []*Job
	for _, j := range all {
		if j.Enabled {
			result = append(result, j)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAtMS < result[j].CreatedAtMS })
	return result
}

// GetJob 按 ID 获取任务详情。
func (s *Service) GetJob(id string) (*Job, bool) {
	return s.store.Get(id)
}

// Status 返回定时任务服务的运行摘要状态。
func (s *Service) Status() map[string]any {
	all := s.store.All()
	enabled := 0
	var nextRun *int64
	for _, j := range all {
		if j.Enabled {
			enabled++
		}
		if j.State.NextRunAtMS != nil {
			if nextRun == nil || *j.State.NextRunAtMS < *nextRun {
				nextRun = j.State.NextRunAtMS
			}
		}
	}

	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	status := map[string]any{
		"running":      running,
		"total_jobs":   len(all),
		"enabled_jobs": enabled,
	}
	if nextRun != nil {
		status["next_run"] = time.UnixMilli(*nextRun).Format(time.RFC3339)
	}
	return status
}
