package bus

import (
	"context"
	"fmt"
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
	SessionID string
	Content   string
	Timestamp time.Time
	Media     []string
	Metadata  map[string]any
	RequestID string
}

// SessionKey returns unique session identifier
func (m *InboundMessage) SessionKey() string {
	if strings.TrimSpace(m.SessionID) != "" {
		return m.SessionID
	}
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

const (
	SystemChannel            = "system"
	SystemTypeSubagentResult = "subagent_result"

	SystemMetaType          = "system_type"
	SystemMetaTaskID        = "task_id"
	SystemMetaTaskLabel     = "task_label"
	SystemMetaOriginChannel = "origin_channel"
	SystemMetaOriginChatID  = "origin_chat_id"
	SystemMetaOriginSender  = "origin_sender_id"
	SystemMetaStatus        = "status"
)

// NewSubagentResultInbound creates a normalized system message for async subagent callbacks.
func NewSubagentResultInbound(taskID, label, originChannel, originChatID, originSenderID, result, requestID string, runErr error) *InboundMessage {
	status := "completed"
	content := strings.TrimSpace(result)
	if runErr != nil {
		status = "failed"
		if content == "" {
			content = runErr.Error()
		} else {
			content = fmt.Sprintf("%s\n\nError: %v", content, runErr)
		}
	}
	if content == "" {
		content = "(empty subagent result)"
	}

	return &InboundMessage{
		Channel:   SystemChannel,
		SenderID:  "subagent:" + strings.TrimSpace(taskID),
		ChatID:    strings.TrimSpace(originChatID),
		Content:   content,
		RequestID: strings.TrimSpace(requestID),
		Metadata: map[string]any{
			SystemMetaType:          SystemTypeSubagentResult,
			SystemMetaTaskID:        strings.TrimSpace(taskID),
			SystemMetaTaskLabel:     strings.TrimSpace(label),
			SystemMetaOriginChannel: strings.TrimSpace(originChannel),
			SystemMetaOriginChatID:  strings.TrimSpace(originChatID),
			SystemMetaOriginSender:  strings.TrimSpace(originSenderID),
			SystemMetaStatus:        status,
		},
	}
}
