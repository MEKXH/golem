package qq

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
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"golang.org/x/oauth2"
)

// Channel implements QQ channel via botgo.
type Channel struct {
	channel.BaseChannel
	cfg            *config.QQConfig
	api            openapi.OpenAPI
	tokenSource    oauth2.TokenSource
	sessionManager botgo.SessionManager

	mu           sync.Mutex
	running      bool
	processedIDs map[string]struct{}
	cancel       context.CancelFunc
}

// New creates QQ channel.
func New(cfg *config.QQConfig, msgBus *bus.MessageBus) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	return &Channel{
		BaseChannel:  channel.BaseChannel{Bus: msgBus, AllowList: allowList},
		cfg:          cfg,
		processedIDs: map[string]struct{}{},
	}
}

func (c *Channel) Name() string { return "qq" }

func (c *Channel) Start(ctx context.Context) error {
	if c.cfg == nil {
		return fmt.Errorf("missing qq config")
	}
	if strings.TrimSpace(c.cfg.AppID) == "" || strings.TrimSpace(c.cfg.AppSecret) == "" {
		return fmt.Errorf("qq app_id and app_secret are required")
	}

	creds := &token.QQBotCredentials{AppID: c.cfg.AppID, AppSecret: c.cfg.AppSecret}
	ts := token.NewQQBotTokenSource(creds)
	runCtx, cancel := context.WithCancel(ctx)

	if err := token.StartRefreshAccessToken(runCtx, ts); err != nil {
		cancel()
		return fmt.Errorf("start qq token refresh: %w", err)
	}

	api := botgo.NewOpenAPI(c.cfg.AppID, ts).WithTimeout(5 * time.Second)
	intent := event.RegisterHandlers(
		c.handleC2CMessage(),
		c.handleGroupATMessage(),
	)
	wsInfo, err := api.WS(runCtx, nil, "")
	if err != nil {
		cancel()
		return fmt.Errorf("get qq websocket info: %w", err)
	}

	sm := botgo.NewSessionManager()
	go func() {
		if err := sm.Start(wsInfo, ts, &intent); err != nil && runCtx.Err() == nil {
			slog.Error("qq websocket stopped", "error", err)
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
		}
	}()

	c.mu.Lock()
	c.api = api
	c.tokenSource = ts
	c.sessionManager = sm
	c.running = true
	c.cancel = cancel
	c.mu.Unlock()

	slog.Info("qq channel started")
	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.running = false
	c.mu.Unlock()
	return nil
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	c.mu.Lock()
	running := c.running
	api := c.api
	c.mu.Unlock()
	if !running || api == nil {
		return fmt.Errorf("qq channel not running")
	}
	if strings.TrimSpace(msg.ChatID) == "" {
		return fmt.Errorf("qq chat id is empty")
	}

	payload := &dto.MessageToCreate{Content: msg.Content}
	if _, err := api.PostC2CMessage(ctx, msg.ChatID, payload); err != nil {
		return fmt.Errorf("send qq message: %w", err)
	}
	return nil
}

func (c *Channel) handleC2CMessage() event.C2CMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
		if data == nil || data.ID == "" || c.isDuplicate(data.ID) {
			return nil
		}
		if data.Author == nil || data.Author.ID == "" {
			return nil
		}
		senderID := data.Author.ID
		if !c.IsAllowed(senderID) {
			return nil
		}
		content := strings.TrimSpace(data.Content)
		if content == "" {
			return nil
		}

		c.PublishInbound(&bus.InboundMessage{
			Channel:   c.Name(),
			SenderID:  senderID,
			ChatID:    senderID,
			Content:   content,
			Timestamp: time.Now(),
			Metadata:  map[string]any{"message_id": data.ID},
			RequestID: bus.NewRequestID(),
		})
		return nil
	}
}

func (c *Channel) handleGroupATMessage() event.GroupATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		if data == nil || data.ID == "" || c.isDuplicate(data.ID) {
			return nil
		}
		if data.Author == nil || data.Author.ID == "" {
			return nil
		}
		senderID := data.Author.ID
		if !c.IsAllowed(senderID) {
			return nil
		}
		content := strings.TrimSpace(data.Content)
		if content == "" {
			return nil
		}

		c.PublishInbound(&bus.InboundMessage{
			Channel:   c.Name(),
			SenderID:  senderID,
			ChatID:    data.GroupID,
			Content:   content,
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"message_id": data.ID,
				"group_id":   data.GroupID,
			},
			RequestID: bus.NewRequestID(),
		})
		return nil
	}
}

func (c *Channel) isDuplicate(messageID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.processedIDs[messageID]; ok {
		return true
	}
	c.processedIDs[messageID] = struct{}{}

	if len(c.processedIDs) > 10000 {
		count := 0
		for id := range c.processedIDs {
			delete(c.processedIDs, id)
			count++
			if count >= 5000 {
				break
			}
		}
	}
	return false
}
