package agent

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/tools"
)

// SubagentTaskRequest defines a delegated subagent run request.
type SubagentTaskRequest struct {
	Task           string
	Label          string
	OriginChannel  string
	OriginChatID   string
	OriginSenderID string
	RequestID      string
}

// SubagentManagerOptions configures timeout/retry/concurrency for delegated tasks.
type SubagentManagerOptions struct {
	Timeout        time.Duration
	Retry          int
	MaxConcurrency int
}

// subagentProcessor is the minimal processing contract used by subagents.
type subagentProcessor interface {
	ProcessForChannelWithSession(ctx context.Context, channel, chatID, senderID, sessionID, content string) (string, error)
}

// SubagentManager executes delegated tasks in background or synchronously.
type SubagentManager struct {
	msgBus    *bus.MessageBus
	processor subagentProcessor
	timeout   time.Duration
	retry     int
	nextID    uint64
	semaphore chan struct{}
	mu        sync.RWMutex
}

// NewSubagentManager creates a subagent manager.
func NewSubagentManager(msgBus *bus.MessageBus, processor subagentProcessor, timeout time.Duration) *SubagentManager {
	return NewSubagentManagerWithOptions(msgBus, processor, SubagentManagerOptions{
		Timeout:        timeout,
		Retry:          1,
		MaxConcurrency: 3,
	})
}

// NewSubagentManagerWithOptions creates a subagent manager with orchestration options.
func NewSubagentManagerWithOptions(msgBus *bus.MessageBus, processor subagentProcessor, options SubagentManagerOptions) *SubagentManager {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	retry := options.Retry
	if retry < 0 {
		retry = 0
	}
	maxConcurrency := options.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 3
	}
	return &SubagentManager{
		msgBus:    msgBus,
		processor: processor,
		timeout:   timeout,
		retry:     retry,
		semaphore: make(chan struct{}, maxConcurrency),
	}
}

// Spawn starts a subagent task asynchronously.
func (m *SubagentManager) Spawn(ctx context.Context, req tools.SubagentRequest) (string, error) {
	normalized, err := m.normalize(req)
	if err != nil {
		return "", err
	}
	taskID := m.nextTaskID()
	go m.run(taskID, normalized)
	return taskID, nil
}

// RunSync executes a subagent task synchronously and returns its result.
func (m *SubagentManager) RunSync(ctx context.Context, req tools.SubagentRequest) (string, error) {
	normalized, err := m.normalize(req)
	if err != nil {
		return "", err
	}
	taskID := m.nextTaskID()
	runCtx, cancel := m.withTimeout(ctx)
	defer cancel()
	return m.executeWithRetry(runCtx, taskID, normalized)
}

// RunWorkflow executes a subagent workflow in sequential or parallel mode and returns an aggregated summary.
func (m *SubagentManager) RunWorkflow(ctx context.Context, req tools.WorkflowRequest) (string, error) {
	normalized, err := m.normalizeWorkflow(req)
	if err != nil {
		return "", err
	}

	runCtx, cancel := m.withTimeout(ctx)
	defer cancel()

	type stepResult struct {
		task   string
		output string
		err    error
	}

	results := make([]stepResult, len(normalized.Subtasks))
	workflowID := m.nextTaskID()

	runStep := func(index int, task string) {
		stepReq := SubagentTaskRequest{
			Task:           task,
			Label:          normalized.Label,
			OriginChannel:  normalized.OriginChannel,
			OriginChatID:   normalized.OriginChatID,
			OriginSenderID: normalized.OriginSenderID,
			RequestID:      normalized.RequestID,
		}
		stepTaskID := fmt.Sprintf("%s-step-%d", workflowID, index+1)
		output, stepErr := m.executeWithRetry(runCtx, stepTaskID, stepReq)
		results[index] = stepResult{task: task, output: output, err: stepErr}
	}

	if normalized.Mode == "parallel" {
		var wg sync.WaitGroup
		for i, task := range normalized.Subtasks {
			wg.Add(1)
			go func(idx int, subtask string) {
				defer wg.Done()
				runStep(idx, subtask)
			}(i, task)
		}
		wg.Wait()
	} else {
		for i, task := range normalized.Subtasks {
			runStep(i, task)
		}
	}

	succeeded := 0
	failed := 0
	var b strings.Builder
	fmt.Fprintf(&b, "Workflow goal: %s\n", normalized.Goal)
	fmt.Fprintf(&b, "Mode: %s total=%d succeeded=%d failed=%d\n", normalized.Mode, len(results), 0, 0)
	for i, result := range results {
		if result.err != nil {
			failed++
			fmt.Fprintf(&b, "[%d] FAIL %s -> %v\n", i+1, result.task, result.err)
			continue
		}
		succeeded++
		fmt.Fprintf(&b, "[%d] OK %s\n%s\n", i+1, result.task, strings.TrimSpace(result.output))
	}
	summary := strings.TrimSpace(b.String())
	summary = strings.Replace(summary, "succeeded=0 failed=0", fmt.Sprintf("succeeded=%d failed=%d", succeeded, failed), 1)
	return summary, nil
}

func (m *SubagentManager) nextTaskID() string {
	id := atomic.AddUint64(&m.nextID, 1)
	return fmt.Sprintf("subagent-%d", id)
}

func (m *SubagentManager) normalize(req tools.SubagentRequest) (SubagentTaskRequest, error) {
	m.mu.RLock()
	processor := m.processor
	m.mu.RUnlock()
	if processor == nil {
		return SubagentTaskRequest{}, fmt.Errorf("subagent processor is not configured")
	}

	task := strings.TrimSpace(req.Task)
	if task == "" {
		return SubagentTaskRequest{}, fmt.Errorf("task is required")
	}

	channel := strings.TrimSpace(req.OriginChannel)
	if channel == "" {
		channel = "cli"
	}
	chatID := strings.TrimSpace(req.OriginChatID)
	if chatID == "" {
		chatID = "direct"
	}
	sender := strings.TrimSpace(req.OriginSenderID)
	if sender == "" {
		sender = "user"
	}

	return SubagentTaskRequest{
		Task:           task,
		Label:          strings.TrimSpace(req.Label),
		OriginChannel:  channel,
		OriginChatID:   chatID,
		OriginSenderID: sender,
		RequestID:      strings.TrimSpace(req.RequestID),
	}, nil
}

func (m *SubagentManager) run(taskID string, req SubagentTaskRequest) {
	baseCtx := context.Background()
	if req.RequestID != "" {
		baseCtx = bus.WithRequestID(baseCtx, req.RequestID)
	}
	ctx, cancel := m.withTimeout(baseCtx)
	defer cancel()

	result, err := m.executeWithRetry(ctx, taskID, req)
	if m.msgBus == nil {
		return
	}

	m.msgBus.PublishInbound(bus.NewSubagentResultInbound(
		taskID,
		req.Label,
		req.OriginChannel,
		req.OriginChatID,
		req.OriginSenderID,
		result,
		req.RequestID,
		err,
	))
}

func (m *SubagentManager) executeOnce(ctx context.Context, taskID string, req SubagentTaskRequest) (string, error) {
	m.mu.RLock()
	processor := m.processor
	m.mu.RUnlock()
	if processor == nil {
		return "", fmt.Errorf("subagent processor is not configured")
	}

	sessionID := "subagent:" + taskID
	senderID := "subagent:" + taskID
	return processor.ProcessForChannelWithSession(
		ctx,
		req.OriginChannel,
		req.OriginChatID,
		senderID,
		sessionID,
		req.Task,
	)
}

func (m *SubagentManager) executeWithRetry(ctx context.Context, taskID string, req SubagentTaskRequest) (string, error) {
	maxAttempts := m.retry + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := m.acquire(ctx); err != nil {
			return "", err
		}
		output, err := m.executeOnce(ctx, taskID, req)
		m.release()
		if err == nil {
			return output, nil
		}
		lastErr = err
		if attempt >= maxAttempts {
			break
		}
		if waitErr := waitRetryBackoff(ctx, attempt); waitErr != nil {
			return "", waitErr
		}
	}
	return "", lastErr
}

func waitRetryBackoff(ctx context.Context, attempt int) error {
	if attempt <= 0 {
		return nil
	}
	delay := time.Duration(math.Min(float64(attempt), 3)) * 150 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (m *SubagentManager) acquire(ctx context.Context) error {
	if m.semaphore == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.semaphore <- struct{}{}:
		return nil
	}
}

func (m *SubagentManager) release() {
	if m.semaphore == nil {
		return
	}
	select {
	case <-m.semaphore:
	default:
	}
}

func (m *SubagentManager) withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if m.timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, m.timeout)
}

func (m *SubagentManager) normalizeWorkflow(req tools.WorkflowRequest) (tools.WorkflowRequest, error) {
	goal := strings.TrimSpace(req.Goal)
	if goal == "" {
		return tools.WorkflowRequest{}, fmt.Errorf("goal is required")
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	switch mode {
	case "", "sequential", "parallel":
	default:
		return tools.WorkflowRequest{}, fmt.Errorf("workflow mode must be one of sequential, parallel; got %q", req.Mode)
	}

	subtasks := make([]string, 0, len(req.Subtasks))
	for _, subtask := range req.Subtasks {
		task := strings.TrimSpace(subtask)
		if task != "" {
			subtasks = append(subtasks, task)
		}
	}
	if len(subtasks) == 0 {
		subtasks = splitWorkflowGoal(goal)
	}
	if len(subtasks) == 0 {
		subtasks = []string{goal}
	}

	if mode == "" {
		if len(subtasks) > 1 {
			mode = "parallel"
		} else {
			mode = "sequential"
		}
	}

	channel := strings.TrimSpace(req.OriginChannel)
	if channel == "" {
		channel = "cli"
	}
	chatID := strings.TrimSpace(req.OriginChatID)
	if chatID == "" {
		chatID = "direct"
	}
	senderID := strings.TrimSpace(req.OriginSenderID)
	if senderID == "" {
		senderID = "user"
	}

	return tools.WorkflowRequest{
		Goal:           goal,
		Mode:           mode,
		Subtasks:       subtasks,
		Label:          strings.TrimSpace(req.Label),
		OriginChannel:  channel,
		OriginChatID:   chatID,
		OriginSenderID: senderID,
		RequestID:      strings.TrimSpace(req.RequestID),
	}, nil
}

func splitWorkflowGoal(goal string) []string {
	replaced := strings.NewReplacer("；", "\n", ";", "\n", "。", "\n", ".", "\n").Replace(goal)
	lines := strings.Split(replaced, "\n")
	out := make([]string, 0, len(lines))
	for _, raw := range lines {
		task := strings.TrimSpace(raw)
		if task != "" {
			out = append(out, task)
		}
	}
	return out
}
