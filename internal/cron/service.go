package cron

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/adhocore/gronx"
)

// JobHandler is called when a job fires.
type JobHandler func(*Job) error

// Service manages scheduled jobs with a ticker-based polling loop.
type Service struct {
	store    *Store
	onJob    JobHandler
	mu       sync.RWMutex
	stopChan chan struct{}
	stopped  chan struct{}
	running  bool
}

// NewService creates a cron service backed by the given store path.
func NewService(storePath string, handler JobHandler) *Service {
	return &Service{
		store: NewStore(storePath),
		onJob: handler,
	}
}

// Start loads jobs from disk and begins the polling loop.
func (s *Service) Start() error {
	if err := s.store.Load(); err != nil {
		return fmt.Errorf("cron service start: %w", err)
	}

	// Compute initial NextRunAtMS for jobs that need it.
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

// Stop gracefully shuts down the polling loop.
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
		if *j.State.NextRunAtMS <= now {
			// Clear NextRunAtMS to prevent re-firing.
			j.State.NextRunAtMS = nil
			s.store.Put(j)
			due = append(due, j)
		}
	}

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

	switch job.Schedule.Kind {
	case "at":
		if job.DeleteAfterRun {
			s.store.Delete(job.ID)
		} else {
			job.Enabled = false
			s.store.Put(job)
		}
	default:
		s.computeNextRun(job)
		s.store.Put(job)
	}

	if err := s.store.Save(); err != nil {
		slog.Warn("cron: failed to save after execution", "error", err)
	}
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

// AddJob creates and persists a new job.
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

// RemoveJob deletes a job by ID.
func (s *Service) RemoveJob(id string) error {
	if !s.store.Delete(id) {
		return fmt.Errorf("job not found: %s", id)
	}
	if err := s.store.Save(); err != nil {
		return fmt.Errorf("save after remove: %w", err)
	}
	return nil
}

// EnableJob sets a job's enabled state.
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

// ListJobs returns all jobs, optionally including disabled ones.
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

// GetJob retrieves a single job by ID.
func (s *Service) GetJob(id string) (*Job, bool) {
	return s.store.Get(id)
}

// Status returns a summary of the cron service.
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
