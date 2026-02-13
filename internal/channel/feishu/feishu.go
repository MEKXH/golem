package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

// Channel implements Feishu bot channel.
type Channel struct {
	channel.BaseChannel
	cfg      *config.FeishuConfig
	client   *lark.Client
	wsClient *larkws.Client

	mu     sync.Mutex
	cancel context.CancelFunc
	run    bool
}

// New creates a Feishu channel.
func New(cfg *config.FeishuConfig, msgBus *bus.MessageBus) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	return &Channel{
		BaseChannel: channel.BaseChannel{Bus: msgBus, AllowList: allowList},
		cfg:         cfg,
		client:      lark.NewClient(cfg.AppID, cfg.AppSecret),
	}
}

func (c *Channel) Name() string { return "feishu" }

func (c *Channel) Start(ctx context.Context) error {
	if c.cfg == nil {
		return fmt.Errorf("missing feishu config")
	}
	if strings.TrimSpace(c.cfg.AppID) == "" || strings.TrimSpace(c.cfg.AppSecret) == "" {
		return fmt.Errorf("feishu app_id/app_secret are required")
	}

	dispatcher := larkdispatcher.NewEventDispatcher(c.cfg.VerificationToken, c.cfg.EncryptKey).
		OnP2MessageReceiveV1(c.handleMessageReceive)

	runCtx, cancel := context.WithCancel(ctx)
	client := larkws.NewClient(
		c.cfg.AppID,
		c.cfg.AppSecret,
		larkws.WithEventHandler(dispatcher),
	)

	c.mu.Lock()
	c.wsClient = client
	c.cancel = cancel
	c.run = true
	c.mu.Unlock()

	go func() {
		if err := client.Start(runCtx); err != nil && runCtx.Err() == nil {
			slog.Error("feishu websocket exited", "error", err)
		}
	}()

	slog.Info("feishu channel started")
	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.wsClient = nil
	c.run = false
	c.mu.Unlock()
	return nil
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	c.mu.Lock()
	running := c.run
	c.mu.Unlock()
	if !running {
		return fmt.Errorf("feishu channel not running")
	}
	if strings.TrimSpace(msg.ChatID) == "" {
		return fmt.Errorf("feishu chat id is empty")
	}

	payload, err := json.Marshal(map[string]string{"text": msg.Content})
	if err != nil {
		return fmt.Errorf("marshal feishu content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(msg.ChatID).
			MsgType(larkim.MsgTypeText).
			Content(string(payload)).
			Uuid(fmt.Sprintf("golem-%d", time.Now().UnixNano())).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("send feishu message: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu api error: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

func (c *Channel) handleMessageReceive(_ context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	sender := event.Event.Sender
	chatID := stringPtrValue(message.ChatId)
	if chatID == "" {
		return nil
	}

	senderID := extractSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}
	if !c.IsAllowed(senderID) {
		return nil
	}

	content := extractMessageContent(message)
	if strings.TrimSpace(content) == "" {
		return nil
	}

	metadata := map[string]any{}
	if id := stringPtrValue(message.MessageId); id != "" {
		metadata["message_id"] = id
	}
	if typ := stringPtrValue(message.MessageType); typ != "" {
		metadata["message_type"] = typ
	}
	if chatType := stringPtrValue(message.ChatType); chatType != "" {
		metadata["chat_type"] = chatType
	}

	c.PublishInbound(&bus.InboundMessage{
		Channel:   c.Name(),
		SenderID:  senderID,
		ChatID:    chatID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  metadata,
		RequestID: bus.NewRequestID(),
	})
	return nil
}

func extractSenderID(sender *larkim.EventSender) string {
	if sender == nil || sender.SenderId == nil {
		return ""
	}
	if sender.SenderId.UserId != nil && *sender.SenderId.UserId != "" {
		return *sender.SenderId.UserId
	}
	if sender.SenderId.OpenId != nil && *sender.SenderId.OpenId != "" {
		return *sender.SenderId.OpenId
	}
	if sender.SenderId.UnionId != nil && *sender.SenderId.UnionId != "" {
		return *sender.SenderId.UnionId
	}
	return ""
}

func extractMessageContent(message *larkim.EventMessage) string {
	if message == nil || message.Content == nil || *message.Content == "" {
		return ""
	}
	if message.MessageType != nil && *message.MessageType == larkim.MsgTypeText {
		var textPayload struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &textPayload); err == nil {
			return textPayload.Text
		}
	}
	return *message.Content
}

func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
