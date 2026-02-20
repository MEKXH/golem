package tools

import (
	"context"
	"strings"
)

type invocationContextKey struct{}

// InvocationContext carries caller metadata for tool execution.
type InvocationContext struct {
	Channel   string
	ChatID    string
	SenderID  string
	RequestID string
	SessionID string
}

// WithInvocationContext stores invocation metadata in context for tools.
func WithInvocationContext(ctx context.Context, meta InvocationContext) context.Context {
	return context.WithValue(ctx, invocationContextKey{}, meta)
}

// InvocationFromContext reads invocation metadata from context.
func InvocationFromContext(ctx context.Context) InvocationContext {
	v := ctx.Value(invocationContextKey{})
	meta, ok := v.(InvocationContext)
	if !ok {
		return InvocationContext{}
	}
	meta.Channel = strings.TrimSpace(meta.Channel)
	meta.ChatID = strings.TrimSpace(meta.ChatID)
	meta.SenderID = strings.TrimSpace(meta.SenderID)
	meta.RequestID = strings.TrimSpace(meta.RequestID)
	meta.SessionID = strings.TrimSpace(meta.SessionID)
	return meta
}
