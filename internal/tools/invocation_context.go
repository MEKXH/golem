package tools

import (
	"context"
	"strings"
)

type invocationContextKey struct{}

// InvocationContext 携带工具执行时的调用者元数据，如通道、聊天 ID 等。
type InvocationContext struct {
	Channel   string // 调用来源通道（如 telegram, cli）
	ChatID    string // 调用来源聊天会话 ID
	SenderID  string // 发起调用的发送者 ID
	RequestID string // 关联的请求追踪 ID
	SessionID string // 关联的会话 ID
}

// WithInvocationContext 将调用元数据注入到 Context 中，供工具逻辑使用。
func WithInvocationContext(ctx context.Context, meta InvocationContext) context.Context {
	return context.WithValue(ctx, invocationContextKey{}, meta)
}

// InvocationFromContext 从 Context 中提取调用元数据。如果 Context 中没有元数据，则返回空结构体。
func InvocationFromContext(ctx context.Context) InvocationContext {
	v := ctx.Value(invocationContextKey{})
	meta, ok := v.(InvocationContext)
	if !ok {
		return InvocationContext{}
	}
	// 确保返回的字符串经过修整，无多余空格
	meta.Channel = strings.TrimSpace(meta.Channel)
	meta.ChatID = strings.TrimSpace(meta.ChatID)
	meta.SenderID = strings.TrimSpace(meta.SenderID)
	meta.RequestID = strings.TrimSpace(meta.RequestID)
	meta.SessionID = strings.TrimSpace(meta.SessionID)
	return meta
}
