package bus

import (
	"context"
	"testing"
)

func TestInboundMessage_SessionKey(t *testing.T) {
	msg := &InboundMessage{
		Channel: "telegram",
		ChatID:  "12345",
	}

	expected := "telegram:12345"
	if got := msg.SessionKey(); got != expected {
		t.Errorf("SessionKey() = %q, want %q", got, expected)
	}
}

func TestInboundMessage_SessionKeyOverride(t *testing.T) {
	msg := &InboundMessage{
		Channel:   "discord",
		ChatID:    "room-1",
		SessionID: "subagent:42",
	}
	if got := msg.SessionKey(); got != "subagent:42" {
		t.Fatalf("expected session override, got %q", got)
	}
}

func TestNewSubagentResultInbound(t *testing.T) {
	msg := NewSubagentResultInbound("task-1", "label", "telegram", "100", "alice", "done", "req-7", nil)
	if msg.Channel != SystemChannel {
		t.Fatalf("expected system channel, got %q", msg.Channel)
	}
	if msg.Metadata[SystemMetaType] != SystemTypeSubagentResult {
		t.Fatalf("unexpected system type metadata: %+v", msg.Metadata)
	}
	if msg.Metadata[SystemMetaOriginChannel] != "telegram" {
		t.Fatalf("unexpected origin channel metadata: %+v", msg.Metadata)
	}
	if msg.Metadata[SystemMetaTaskID] != "task-1" {
		t.Fatalf("unexpected task id metadata: %+v", msg.Metadata)
	}
	if msg.RequestID != "req-7" {
		t.Fatalf("expected request id propagation, got %q", msg.RequestID)
	}
}

func TestRequestIDContext(t *testing.T) {
	ctx := context.Background()
	if got := RequestIDFromContext(ctx); got != "" {
		t.Fatalf("expected empty request id, got %q", got)
	}

	ctx = WithRequestID(ctx, "req-123")
	if got := RequestIDFromContext(ctx); got != "req-123" {
		t.Fatalf("expected req-123, got %q", got)
	}
}
