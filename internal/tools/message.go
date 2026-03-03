package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// MessageInput 定义了 message 工具的输入参数。
type MessageInput struct {
	Content string `json:"content" jsonschema:"required,description=Message content to send"`
	Channel string `json:"channel,omitempty" jsonschema:"description=Target channel (optional; defaults to current channel)"`
	ChatID  string `json:"chat_id,omitempty" jsonschema:"description=Target chat/session id (optional; defaults to current chat)"`
}

type messageToolImpl struct {
	publisher interface {
		PublishOutbound(msg *bus.OutboundMessage)
	}
}

func (t *messageToolImpl) execute(ctx context.Context, input *MessageInput) (string, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return "", fmt.Errorf("content is required")
	}
	if t.publisher == nil {
		return "", fmt.Errorf("message publisher is not configured")
	}

	meta := InvocationFromContext(ctx)
	channel := strings.TrimSpace(input.Channel)
	chatID := strings.TrimSpace(input.ChatID)
	// 如果未指定通道或聊天 ID，则回退到当前调用的上下文
	if channel == "" {
		channel = meta.Channel
	}
	if chatID == "" {
		chatID = meta.ChatID
	}
	if channel == "" || chatID == "" {
		return "", fmt.Errorf("channel/chat_id is required when no invocation context is available")
	}

	reqID := meta.RequestID
	if reqID == "" {
		reqID = bus.NewRequestID()
	}

	t.publisher.PublishOutbound(&bus.OutboundMessage{
		Channel:   channel,
		ChatID:    chatID,
		Content:   content,
		RequestID: reqID,
		Metadata: map[string]any{
			"via_tool": "message",
		},
	})

	return fmt.Sprintf("Message sent to %s:%s", channel, chatID), nil
}

// NewMessageTool 创建一个允许 Agent 发送主动消息的工具实例。
func NewMessageTool(publisher interface {
	PublishOutbound(msg *bus.OutboundMessage)
}) (tool.InvokableTool, error) {
	impl := &messageToolImpl{publisher: publisher}
	return utils.InferTool(
		"message",
		"Send a direct message to a channel/chat. Defaults to the current conversation when channel/chat_id is omitted.",
		impl.execute,
	)
}
