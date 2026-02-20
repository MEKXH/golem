package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
	"github.com/gorilla/websocket"
)

// Channel implements a WhatsApp bridge channel over websocket.
type Channel struct {
	channel.BaseChannel
	cfg       *config.WhatsAppConfig
	conn      *websocket.Conn
	mu        sync.RWMutex
	running   bool
	cancelRun context.CancelFunc
}

// New creates a WhatsApp channel instance.
func New(cfg *config.WhatsAppConfig, msgBus *bus.MessageBus) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	return &Channel{
		BaseChannel: channel.BaseChannel{Bus: msgBus, AllowList: allowList},
		cfg:         cfg,
	}
}

func (c *Channel) Name() string { return "whatsapp" }

func (c *Channel) Start(ctx context.Context) error {
	if c.cfg == nil {
		return fmt.Errorf("missing whatsapp config")
	}
	if c.cfg.BridgeURL == "" {
		return fmt.Errorf("whatsapp bridge_url is empty")
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(c.cfg.BridgeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to whatsapp bridge: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.conn = conn
	c.running = true
	c.cancelRun = cancel
	c.mu.Unlock()

	go c.listen(runCtx)
	slog.Info("whatsapp bridge connected")
	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancelRun != nil {
		c.cancelRun()
		c.cancelRun = nil
	}
	conn := c.conn
	c.conn = nil
	c.running = false
	c.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	return nil
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	c.mu.RLock()
	conn := c.conn
	running := c.running
	c.mu.RUnlock()
	if !running || conn == nil {
		return fmt.Errorf("whatsapp channel not running")
	}

	payload := map[string]any{
		"type":    "message",
		"to":      msg.ChatID,
		"content": msg.Content,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal whatsapp message: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("send whatsapp message: %w", err)
	}
	return nil
}

func (c *Channel) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()
		if conn == nil {
			time.Sleep(time.Second)
			continue
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			slog.Warn("whatsapp read failed", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}

		var inbound map[string]any
		if err := json.Unmarshal(raw, &inbound); err != nil {
			slog.Warn("whatsapp decode failed", "error", err)
			continue
		}

		if t, _ := inbound["type"].(string); t != "message" {
			continue
		}

		senderID, _ := inbound["from"].(string)
		if senderID == "" {
			continue
		}
		chatID, _ := inbound["chat"].(string)
		if chatID == "" {
			chatID = senderID
		}
		content, _ := inbound["content"].(string)
		if content == "" {
			continue
		}

		metadata := map[string]any{}
		if messageID, ok := inbound["id"].(string); ok && messageID != "" {
			metadata["message_id"] = messageID
		}
		if userName, ok := inbound["from_name"].(string); ok && userName != "" {
			metadata["username"] = userName
		}

		media := []string{}
		if mediaItems, ok := inbound["media"].([]any); ok {
			for _, item := range mediaItems {
				if path, ok := item.(string); ok && path != "" {
					media = append(media, path)
				}
			}
		}

		c.PublishInbound(&bus.InboundMessage{
			Channel:   c.Name(),
			SenderID:  senderID,
			ChatID:    chatID,
			Content:   content,
			Timestamp: time.Now(),
			Media:     media,
			Metadata:  metadata,
			RequestID: bus.NewRequestID(),
		})
	}
}
