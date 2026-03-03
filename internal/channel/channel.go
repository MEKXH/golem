// Package channel 定义了 Golem 与不同聊天平台（如 Telegram、飞书等）交互的接口和基础实现。
package channel

import (
	"context"
	"strings"

	"github.com/MEKXH/golem/internal/bus"
)

// Channel 定义了聊天平台对接需要实现的接口。
type Channel interface {
	Name() string                                             // 返回通道名称（如 "telegram"）
	Start(ctx context.Context) error                          // 启动通道连接或监听
	Stop(ctx context.Context) error                           // 停止通道连接
	Send(ctx context.Context, msg *bus.OutboundMessage) error // 发送出站消息到平台
	IsAllowed(senderID string) bool                           // 检查指定的发送者是否有权限使用此通道
}

// BaseChannel 提供跨不同通道共享的基础功能。
type BaseChannel struct {
	Bus       *bus.MessageBus // 关联的消息总线，用于转发入站消息
	AllowList map[string]bool // 允许访问此通道的用户 ID 列表（为空则不限制）
}

// IsAllowed 检查发送者 ID 是否在允许名单中。
func (b *BaseChannel) IsAllowed(senderID string) bool {
	if len(b.AllowList) == 0 {
		return true
	}

	idPart := senderID
	userPart := ""
	if idx := strings.Index(senderID, "|"); idx > 0 {
		idPart = senderID[:idx]
		userPart = senderID[idx+1:]
	}

	for allowed := range b.AllowList {
		normalized := strings.TrimSpace(allowed)
		trimmed := strings.TrimPrefix(normalized, "@")
		if normalized == senderID || trimmed == senderID ||
			normalized == idPart || trimmed == idPart ||
			(userPart != "" && (normalized == userPart || trimmed == userPart)) {
			return true
		}
	}

	return false
}

// PublishInbound 将接收到的原始平台消息发布到消息总线中。
func (b *BaseChannel) PublishInbound(msg *bus.InboundMessage) {
	b.Bus.PublishInbound(msg)
}
