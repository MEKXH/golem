package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type phase5E2EModel struct {
	mu sync.Mutex
}

func (m *phase5E2EModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lastUser := ""
	hasToolResult := false
	for _, msg := range input {
		if msg.Role == schema.User {
			lastUser = strings.TrimSpace(msg.Content)
		}
		if msg.Role == schema.Tool {
			hasToolResult = true
		}
	}

	switch lastUser {
	case "run message e2e":
		if !hasToolResult {
			return &schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{
						ID: "tc-message-1",
						Function: schema.FunctionCall{
							Name:      "message",
							Arguments: `{"content":"hello from tool"}`,
						},
					},
				},
			}, nil
		}
		return &schema.Message{Role: schema.Assistant, Content: "done"}, nil

	case "run spawn e2e":
		if !hasToolResult {
			return &schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{
						ID: "tc-spawn-1",
						Function: schema.FunctionCall{
							Name:      "spawn",
							Arguments: `{"task":"subtask payload","label":"diag"}`,
						},
					},
				},
			}, nil
		}
		return &schema.Message{Role: schema.Assistant, Content: "spawn accepted"}, nil

	case "subtask payload":
		return &schema.Message{Role: schema.Assistant, Content: "subagent completed result"}, nil

	default:
		return &schema.Message{Role: schema.Assistant, Content: "ok"}, nil
	}
}

func (m *phase5E2EModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *phase5E2EModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func newPhase5Loop(t *testing.T) (*Loop, context.Context, context.CancelFunc) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	msgBus := bus.NewMessageBus(32)
	chatModel := &phase5E2EModel{}

	loop, err := NewLoop(cfg, msgBus, chatModel)
	if err != nil {
		t.Fatalf("NewLoop: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return loop, ctx, cancel
}

func waitOutbound(t *testing.T, out <-chan *bus.OutboundMessage, timeout time.Duration, match func(*bus.OutboundMessage) bool) *bus.OutboundMessage {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-out:
			if msg != nil && match(msg) {
				return msg
			}
		case <-deadline:
			t.Fatal("timed out waiting for outbound message")
		}
	}
}

func TestPhase5E2E_MessageToolRoute(t *testing.T) {
	loop, ctx, cancel := newPhase5Loop(t)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- loop.Run(ctx) }()

	loop.bus.PublishInbound(&bus.InboundMessage{
		Channel:   "telegram",
		ChatID:    "chat-msg",
		SenderID:  "alice",
		Content:   "run message e2e",
		RequestID: "req-msg-1",
	})

	toolMsg := waitOutbound(t, loop.bus.Outbound(), 3*time.Second, func(m *bus.OutboundMessage) bool {
		return strings.Contains(m.Content, "hello from tool")
	})
	if toolMsg.Channel != "telegram" || toolMsg.ChatID != "chat-msg" {
		t.Fatalf("unexpected message tool route: %+v", toolMsg)
	}
	if toolMsg.RequestID != "req-msg-1" {
		t.Fatalf("expected request id propagation, got %q", toolMsg.RequestID)
	}

	finalMsg := waitOutbound(t, loop.bus.Outbound(), 3*time.Second, func(m *bus.OutboundMessage) bool {
		return strings.Contains(m.Content, "done")
	})
	if finalMsg.Channel != "telegram" || finalMsg.ChatID != "chat-msg" {
		t.Fatalf("unexpected final route: %+v", finalMsg)
	}

	cancel()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected context canceled from Run, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for loop shutdown")
	}
}

func TestPhase5E2E_SpawnSystemCallbackRoute(t *testing.T) {
	loop, ctx, cancel := newPhase5Loop(t)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- loop.Run(ctx) }()

	loop.bus.PublishInbound(&bus.InboundMessage{
		Channel:   "discord",
		ChatID:    "chat-spawn",
		SenderID:  "bob",
		Content:   "run spawn e2e",
		RequestID: "req-spawn-1",
	})

	_ = waitOutbound(t, loop.bus.Outbound(), 3*time.Second, func(m *bus.OutboundMessage) bool {
		return strings.Contains(m.Content, "spawn accepted")
	})

	callbackMsg := waitOutbound(t, loop.bus.Outbound(), 4*time.Second, func(m *bus.OutboundMessage) bool {
		return strings.Contains(m.Content, "subagent completed result")
	})
	if callbackMsg.Channel != "discord" || callbackMsg.ChatID != "chat-spawn" {
		t.Fatalf("unexpected callback route: %+v", callbackMsg)
	}
	if callbackMsg.RequestID != "req-spawn-1" {
		t.Fatalf("expected callback request id propagation, got %q", callbackMsg.RequestID)
	}

	cancel()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected context canceled from Run, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for loop shutdown")
	}
}
