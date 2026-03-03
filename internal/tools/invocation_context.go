package tools

import (
	"context"
	"strings"
)

type invocationContextKey struct{}

// InvocationContext 携带工具执行的调用者元数据。
type InvocationContext struct {
	Channel   string
	ChatID    string
	SenderID  string
	RequestID string
	SessionID string
}

// WithInvocationContext 将调用元数据存储到上下文中供工具使用。
func WithInvocationContext(ctx context.Context, meta InvocationContext) context.Context {
	return context.WithValue(ctx, invocationContextKey{}, meta)
}

// InvocationFromContext 从上下文中读取调用元数据。
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
