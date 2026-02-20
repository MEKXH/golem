package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/tools"
)

type fakeSubagentProcessor struct {
	response string
	err      error
	lastCall struct {
		channel   string
		chatID    string
		senderID  string
		sessionID string
		content   string
	}
}

func (f *fakeSubagentProcessor) ProcessForChannelWithSession(ctx context.Context, channel, chatID, senderID, sessionID, content string) (string, error) {
	f.lastCall.channel = channel
	f.lastCall.chatID = chatID
	f.lastCall.senderID = senderID
	f.lastCall.sessionID = sessionID
	f.lastCall.content = content
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

func TestSubagentManager_SpawnPublishesSystemResult(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	processor := &fakeSubagentProcessor{response: "task done"}
	manager := NewSubagentManager(msgBus, processor, 2*time.Second)

	taskID, err := manager.Spawn(context.Background(), tools.SubagentRequest{
		Task:           "collect diagnostics",
		Label:          "diag",
		OriginChannel:  "telegram",
		OriginChatID:   "1001",
		OriginSenderID: "bob",
		RequestID:      "req-9",
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if taskID == "" {
		t.Fatal("expected non-empty task id")
	}

	select {
	case msg := <-msgBus.Inbound():
		if msg.Channel != bus.SystemChannel {
			t.Fatalf("expected system channel, got %q", msg.Channel)
		}
		if msg.Metadata[bus.SystemMetaType] != bus.SystemTypeSubagentResult {
			t.Fatalf("unexpected system type metadata: %+v", msg.Metadata)
		}
		if msg.Metadata[bus.SystemMetaOriginChannel] != "telegram" || msg.Metadata[bus.SystemMetaOriginChatID] != "1001" {
			t.Fatalf("unexpected origin metadata: %+v", msg.Metadata)
		}
		if msg.Metadata[bus.SystemMetaTaskID] == "" {
			t.Fatalf("expected task id metadata, got %+v", msg.Metadata)
		}
		if msg.RequestID != "req-9" {
			t.Fatalf("expected request id propagation, got %q", msg.RequestID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subagent system message")
	}
}

func TestSubagentManager_RunSyncReturnsError(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	processor := &fakeSubagentProcessor{err: errors.New("boom")}
	manager := NewSubagentManager(msgBus, processor, 2*time.Second)

	_, err := manager.RunSync(context.Background(), tools.SubagentRequest{
		Task:           "failing task",
		OriginChannel:  "discord",
		OriginChatID:   "x",
		OriginSenderID: "alice",
	})
	if err == nil {
		t.Fatal("expected sync run error")
	}
}

type flakySubagentProcessor struct {
	mu        sync.Mutex
	failFirst int
	calls     int
}

func (f *flakySubagentProcessor) ProcessForChannelWithSession(ctx context.Context, channel, chatID, senderID, sessionID, content string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls++
	if f.calls <= f.failFirst {
		return "", fmt.Errorf("transient error #%d", f.calls)
	}
	return "ok:" + strings.TrimSpace(content), nil
}

func TestSubagentManager_RunSyncRetriesTransientFailure(t *testing.T) {
	processor := &flakySubagentProcessor{failFirst: 1}
	manager := NewSubagentManagerWithOptions(nil, processor, SubagentManagerOptions{
		Timeout:        2 * time.Second,
		Retry:          1,
		MaxConcurrency: 2,
	})

	out, err := manager.RunSync(context.Background(), tools.SubagentRequest{
		Task:           "retry me",
		OriginChannel:  "cli",
		OriginChatID:   "direct",
		OriginSenderID: "user",
	})
	if err != nil {
		t.Fatalf("RunSync should recover with retry, got err: %v", err)
	}
	if !strings.Contains(out, "ok:retry me") {
		t.Fatalf("unexpected RunSync output: %q", out)
	}
	if processor.calls != 2 {
		t.Fatalf("expected two attempts, got %d", processor.calls)
	}
}

type concurrencyProbeProcessor struct {
	mu        sync.Mutex
	active    int
	maxActive int
	delay     time.Duration
}

func (p *concurrencyProbeProcessor) ProcessForChannelWithSession(ctx context.Context, channel, chatID, senderID, sessionID, content string) (string, error) {
	p.mu.Lock()
	p.active++
	if p.active > p.maxActive {
		p.maxActive = p.active
	}
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
		return "", ctx.Err()
	case <-time.After(p.delay):
	}

	p.mu.Lock()
	p.active--
	p.mu.Unlock()
	return "done", nil
}

func (p *concurrencyProbeProcessor) MaxActive() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.maxActive
}

func TestSubagentManager_MaxConcurrencyCapsParallelRuns(t *testing.T) {
	processor := &concurrencyProbeProcessor{delay: 120 * time.Millisecond}
	manager := NewSubagentManagerWithOptions(nil, processor, SubagentManagerOptions{
		Timeout:        2 * time.Second,
		Retry:          0,
		MaxConcurrency: 1,
	})

	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := manager.RunSync(context.Background(), tools.SubagentRequest{
				Task:           fmt.Sprintf("task-%d", idx),
				OriginChannel:  "cli",
				OriginChatID:   "direct",
				OriginSenderID: "user",
			})
			errs <- err
		}(i + 1)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("RunSync unexpected error: %v", err)
		}
	}
	if got := processor.MaxActive(); got != 1 {
		t.Fatalf("expected max active workers=1, got %d", got)
	}
}

type workflowTestProcessor struct{}

func (p *workflowTestProcessor) ProcessForChannelWithSession(ctx context.Context, channel, chatID, senderID, sessionID, content string) (string, error) {
	task := strings.TrimSpace(content)
	if strings.Contains(task, "fail") {
		return "", fmt.Errorf("failed subtask: %s", task)
	}
	return "done:" + task, nil
}

func TestSubagentManager_RunWorkflow_ReportsPerTaskFailures(t *testing.T) {
	manager := NewSubagentManagerWithOptions(nil, &workflowTestProcessor{}, SubagentManagerOptions{
		Timeout:        2 * time.Second,
		Retry:          0,
		MaxConcurrency: 2,
	})

	out, err := manager.RunWorkflow(context.Background(), tools.WorkflowRequest{
		Goal:           "deploy verification",
		Mode:           "parallel",
		Subtasks:       []string{"check health", "fail smoke", "collect metrics"},
		OriginChannel:  "cli",
		OriginChatID:   "direct",
		OriginSenderID: "user",
	})
	if err != nil {
		t.Fatalf("RunWorkflow should return summary (not hard fail), got err: %v", err)
	}

	lower := strings.ToLower(out)
	if !strings.Contains(lower, "failed=1") {
		t.Fatalf("expected one failed subtask in summary, got: %s", out)
	}
	if !strings.Contains(lower, "fail smoke") {
		t.Fatalf("expected failed subtask name in summary, got: %s", out)
	}
}
