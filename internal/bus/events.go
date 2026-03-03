// Package bus 实现 Golem 的消息总线机制，支持通道与 Agent 之间的异步通信。
package bus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type requestIDContextKey struct{}

// InboundMessage 表示从外部通道（如 Telegram、飞书等）接收到的入站消息。
type InboundMessage struct {
	Channel   string         // 消息来源通道（如 "telegram"）
	SenderID  string         // 发送者唯一 ID
	ChatID    string         // 聊天会话 ID
	SessionID string         // 显式指定的会话 ID（可选）
	Content   string         // 消息文本内容
	Timestamp time.Time      // 接收时间
	Media     []string       // 附件（如图片 URL）列表
	Metadata  map[string]any // 随消息携带的元数据
	RequestID string         // 用于追踪的请求 ID
}

// SessionKey 返回此消息对应的唯一会话标识符。
func (m *InboundMessage) SessionKey() string {
	if strings.TrimSpace(m.SessionID) != "" {
		return m.SessionID
	}
	return m.Channel + ":" + m.ChatID
}

// OutboundMessage 表示发送给外部通道的出站消息。
type OutboundMessage struct {
	Channel   string         // 目标通道
	ChatID    string         // 目标聊天 ID
	Content   string         // 消息文本内容
	ReplyTo   string         // 回复的消息 ID（可选）
	Media     []string       // 待发送的媒体文件列表
	Metadata  map[string]any // 随消息携带的元数据
	RequestID string         // 关联的请求 ID
}

// NewRequestID 生成一个新的 UUID 用于请求追踪。
func NewRequestID() string {
	return uuid.NewString()
}

// WithRequestID 将请求 ID 注入到 context 中。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext 从 context 中读取请求 ID。
func RequestIDFromContext(ctx context.Context) string {
	v := ctx.Value(requestIDContextKey{})
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

const (
	SystemChannel            = "system"          // 系统内部专用通道
	SystemTypeSubagentResult = "subagent_result" // 系统消息类型：子 Agent 执行结果

	SystemMetaType          = "system_type"      // 元数据键：消息类型
	SystemMetaTaskID        = "task_id"          // 元数据键：任务 ID
	SystemMetaTaskLabel     = "task_label"       // 元数据键：任务标签
	SystemMetaOriginChannel = "origin_channel"   // 元数据键：原始请求通道
	SystemMetaOriginChatID  = "origin_chat_id"   // 元数据键：原始请求聊天 ID
	SystemMetaOriginSender  = "origin_sender_id" // 元数据键：原始发送者 ID
	SystemMetaStatus        = "status"           // 元数据键：执行状态
)

// NewSubagentResultInbound 为异步子 Agent 回调创建一个规范化的系统入站消息。
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
