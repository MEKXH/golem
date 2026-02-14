package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/bus"
)

type capturePublisher struct {
	msgs []*bus.OutboundMessage
}

func (p *capturePublisher) PublishOutbound(msg *bus.OutboundMessage) {
	p.msgs = append(p.msgs, msg)
}

func TestMessageTool_UsesInvocationDefaults(t *testing.T) {
	pub := &capturePublisher{}
	msgTool, err := NewMessageTool(pub)
	if err != nil {
		t.Fatalf("NewMessageTool: %v", err)
	}

	ctx := WithInvocationContext(context.Background(), InvocationContext{
		Channel:   "telegram",
		ChatID:    "123",
		RequestID: "req-1",
	})

	result, err := msgTool.InvokableRun(ctx, `{"content":"hello"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if !strings.Contains(result, "Message sent") {
		t.Fatalf("expected sent confirmation, got: %s", result)
	}
	if len(pub.msgs) != 1 {
		t.Fatalf("expected 1 outbound message, got %d", len(pub.msgs))
	}
	if pub.msgs[0].Channel != "telegram" || pub.msgs[0].ChatID != "123" {
		t.Fatalf("unexpected outbound route: %+v", pub.msgs[0])
	}
	if pub.msgs[0].RequestID != "req-1" {
		t.Fatalf("expected request id to propagate, got %q", pub.msgs[0].RequestID)
	}
}

func TestMessageTool_ExplicitTargetOverridesDefaults(t *testing.T) {
	pub := &capturePublisher{}
	msgTool, err := NewMessageTool(pub)
	if err != nil {
		t.Fatalf("NewMessageTool: %v", err)
	}

	ctx := WithInvocationContext(context.Background(), InvocationContext{
		Channel: "telegram",
		ChatID:  "123",
	})

	_, err = msgTool.InvokableRun(ctx, `{"content":"hello","channel":"discord","chat_id":"abc"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if len(pub.msgs) != 1 {
		t.Fatalf("expected 1 outbound message, got %d", len(pub.msgs))
	}
	if pub.msgs[0].Channel != "discord" || pub.msgs[0].ChatID != "abc" {
		t.Fatalf("expected explicit channel/chat_id override, got %+v", pub.msgs[0])
	}
}

func TestMessageTool_MissingTargetReturnsError(t *testing.T) {
	pub := &capturePublisher{}
	msgTool, err := NewMessageTool(pub)
	if err != nil {
		t.Fatalf("NewMessageTool: %v", err)
	}

	if _, err := msgTool.InvokableRun(context.Background(), `{"content":"hello"}`); err == nil {
		t.Fatal("expected error when no channel/chat can be resolved")
	}
}
