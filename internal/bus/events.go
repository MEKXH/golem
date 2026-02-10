package bus

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

type requestIDContextKey struct{}

// InboundMessage received from a channel
type InboundMessage struct {
	Channel   string
	SenderID  string
	ChatID    string
	Content   string
	Timestamp time.Time
	Media     []string
	Metadata  map[string]any
	RequestID string
}

// SessionKey returns unique session identifier
func (m *InboundMessage) SessionKey() string {
	return m.Channel + ":" + m.ChatID
}

// OutboundMessage to send to a channel
type OutboundMessage struct {
	Channel   string
	ChatID    string
	Content   string
	ReplyTo   string
	Media     []string
	Metadata  map[string]any
	RequestID string
}

// NewRequestID creates a request id for tracing.
func NewRequestID() string {
	return uuid.NewString()
}

// WithRequestID adds a request id to context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext reads request id from context.
func RequestIDFromContext(ctx context.Context) string {
	v := ctx.Value(requestIDContextKey{})
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
