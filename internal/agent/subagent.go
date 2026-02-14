package agent

import (
	"context"
	"fmt"
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

// subagentProcessor is the minimal processing contract used by subagents.
type subagentProcessor interface {
	ProcessForChannelWithSession(ctx context.Context, channel, chatID, senderID, sessionID, content string) (string, error)
}

// SubagentManager executes delegated tasks in background or synchronously.
type SubagentManager struct {
	msgBus    *bus.MessageBus
	processor subagentProcessor
	timeout   time.Duration
	nextID    uint64
	mu        sync.RWMutex
}

// NewSubagentManager creates a subagent manager.
func NewSubagentManager(msgBus *bus.MessageBus, processor subagentProcessor, timeout time.Duration) *SubagentManager {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &SubagentManager{
		msgBus:    msgBus,
		processor: processor,
		timeout:   timeout,
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
	return m.executeOnce(ctx, taskID, normalized)
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
	ctx, cancel := context.WithTimeout(baseCtx, m.timeout)
	defer cancel()

	result, err := m.executeOnce(ctx, taskID, req)
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
