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
