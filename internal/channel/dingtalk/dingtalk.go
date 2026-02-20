package dingtalk

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
)

// Channel implements DingTalk channel via Stream Mode.
type Channel struct {
	channel.BaseChannel
	cfg          *config.DingTalkConfig
	streamClient *client.StreamClient

	mu              sync.RWMutex
	running         bool
	cancel          context.CancelFunc
	sessionWebhooks sync.Map
}

// New creates a DingTalk channel.
func New(cfg *config.DingTalkConfig, msgBus *bus.MessageBus) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	return &Channel{
		BaseChannel: channel.BaseChannel{Bus: msgBus, AllowList: allowList},
		cfg:         cfg,
	}
}

func (c *Channel) Name() string { return "dingtalk" }

func (c *Channel) Start(ctx context.Context) error {
	if c.cfg == nil {
		return fmt.Errorf("missing dingtalk config")
	}
	if strings.TrimSpace(c.cfg.ClientID) == "" || strings.TrimSpace(c.cfg.ClientSecret) == "" {
		return fmt.Errorf("dingtalk client_id/client_secret are required")
	}

	runCtx, cancel := context.WithCancel(ctx)
	cred := client.NewAppCredentialConfig(c.cfg.ClientID, c.cfg.ClientSecret)
	streamClient := client.NewStreamClient(
		client.WithAppCredential(cred),
		client.WithAutoReconnect(true),
	)
	streamClient.RegisterChatBotCallbackRouter(c.onChatBotMessageReceived)

	c.mu.Lock()
	c.streamClient = streamClient
	c.running = true
	c.cancel = cancel
	c.mu.Unlock()

	go func() {
		if err := streamClient.Start(runCtx); err != nil && runCtx.Err() == nil {
			slog.Error("dingtalk stream exited", "error", err)
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
		}
	}()

	slog.Info("dingtalk channel started")
	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	streamClient := c.streamClient
	c.streamClient = nil
	c.running = false
	c.mu.Unlock()

	if streamClient != nil {
		streamClient.Close()
	}
	return nil
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	c.mu.RLock()
	running := c.running
	c.mu.RUnlock()
	if !running {
		return fmt.Errorf("dingtalk channel not running")
	}

	rawWebhook, ok := c.sessionWebhooks.Load(msg.ChatID)
	if !ok {
		return fmt.Errorf("no dingtalk session_webhook for chat %s", msg.ChatID)
	}
	sessionWebhook, ok := rawWebhook.(string)
	if !ok || sessionWebhook == "" {
		return fmt.Errorf("invalid dingtalk session_webhook for chat %s", msg.ChatID)
	}

	replier := chatbot.NewChatbotReplier()
	title := []byte("Golem")
	content := []byte(msg.Content)
	if err := replier.SimpleReplyMarkdown(ctx, sessionWebhook, title, content); err != nil {
		return fmt.Errorf("send dingtalk message: %w", err)
	}
	return nil
}

func (c *Channel) onChatBotMessageReceived(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	content := strings.TrimSpace(data.Text.Content)
	if content == "" {
		if contentMap, ok := data.Content.(map[string]any); ok {
			if text, ok := contentMap["content"].(string); ok {
				content = strings.TrimSpace(text)
			}
		}
	}
	if content == "" {
		return nil, nil
	}

	senderID := data.SenderStaffId
	if senderID == "" {
		return nil, nil
	}
	if !c.IsAllowed(senderID) {
		return nil, nil
	}

	chatID := senderID
	if data.ConversationType != "1" && data.ConversationId != "" {
		chatID = data.ConversationId
	}
	if data.SessionWebhook != "" {
		c.sessionWebhooks.Store(chatID, data.SessionWebhook)
	}

	c.PublishInbound(&bus.InboundMessage{
		Channel:   c.Name(),
		SenderID:  senderID,
		ChatID:    chatID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"sender_name":       data.SenderNick,
			"conversation_id":   data.ConversationId,
			"conversation_type": data.ConversationType,
		},
		RequestID: bus.NewRequestID(),
	})

	return nil, nil
}
