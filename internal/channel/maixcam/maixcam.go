package maixcam

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
)

type cameraMessage struct {
	Type      string         `json:"type"`
	Timestamp float64        `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// Channel implements the MaixCam TCP bridge.
type Channel struct {
	channel.BaseChannel
	cfg      *config.MaixCamConfig
	listener net.Listener
	clients  map[net.Conn]struct{}
	mu       sync.RWMutex
	running  bool
}

// New creates a MaixCam channel.
func New(cfg *config.MaixCamConfig, msgBus *bus.MessageBus) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	return &Channel{
		BaseChannel: channel.BaseChannel{Bus: msgBus, AllowList: allowList},
		cfg:         cfg,
		clients:     map[net.Conn]struct{}{},
	}
}

func (c *Channel) Name() string { return "maixcam" }

func (c *Channel) Start(ctx context.Context) error {
	if c.cfg == nil {
		return fmt.Errorf("missing maixcam config")
	}
	if c.cfg.Host == "" {
		return fmt.Errorf("maixcam host is empty")
	}
	if c.cfg.Port <= 0 {
		return fmt.Errorf("maixcam port must be > 0")
	}

	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("start maixcam listener: %w", err)
	}

	c.mu.Lock()
	c.listener = ln
	c.running = true
	c.mu.Unlock()

	slog.Info("maixcam channel listening", "addr", addr)
	go c.acceptLoop(ctx)
	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	listener := c.listener
	c.listener = nil
	c.running = false
	clients := make([]net.Conn, 0, len(c.clients))
	for conn := range c.clients {
		clients = append(clients, conn)
	}
	c.clients = map[net.Conn]struct{}{}
	c.mu.Unlock()

	if listener != nil {
		_ = listener.Close()
	}
	for _, conn := range clients {
		_ = conn.Close()
	}
	return nil
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	c.mu.RLock()
	if !c.running {
		c.mu.RUnlock()
		return fmt.Errorf("maixcam channel not running")
	}
	clients := make([]net.Conn, 0, len(c.clients))
	for conn := range c.clients {
		clients = append(clients, conn)
	}
	c.mu.RUnlock()

	if len(clients) == 0 {
		return fmt.Errorf("no connected maixcam clients")
	}

	payload, err := json.Marshal(map[string]any{
		"type":      "command",
		"chat_id":   msg.ChatID,
		"message":   msg.Content,
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		return fmt.Errorf("marshal maixcam outbound: %w", err)
	}

	var firstErr error
	for _, conn := range clients {
		if _, err := conn.Write(payload); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			slog.Warn("send maixcam message failed", "remote", conn.RemoteAddr().String(), "error", err)
		}
	}
	if firstErr != nil {
		return fmt.Errorf("send maixcam message: %w", firstErr)
	}
	return nil
}

func (c *Channel) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		ln := c.listener
		running := c.running
		c.mu.RUnlock()
		if !running || ln == nil {
			return
		}

		conn, err := ln.Accept()
		if err != nil {
			if running {
				slog.Warn("accept maixcam connection failed", "error", err)
			}
			return
		}

		c.mu.Lock()
		c.clients[conn] = struct{}{}
		c.mu.Unlock()
		go c.handleConn(ctx, conn)
	}
}

func (c *Channel) handleConn(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
		c.mu.Lock()
		delete(c.clients, conn)
		c.mu.Unlock()
	}()

	decoder := json.NewDecoder(conn)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var msg cameraMessage
		if err := decoder.Decode(&msg); err != nil {
			return
		}

		if msg.Type == "heartbeat" {
			continue
		}
		if msg.Type == "status" {
			slog.Debug("maixcam status", "data", msg.Data)
			continue
		}
		if msg.Type != "person_detected" {
			continue
		}

		className, _ := msg.Data["class_name"].(string)
		if className == "" {
			className = "person"
		}
		score, _ := msg.Data["score"].(float64)
		content := fmt.Sprintf("Person detected: %s (%.2f%%)", className, score*100)
		metadata := map[string]any{"event": msg.Type, "raw": msg.Data}

		c.PublishInbound(&bus.InboundMessage{
			Channel:   c.Name(),
			SenderID:  "maixcam",
			ChatID:    "default",
			Content:   content,
			Timestamp: time.Now(),
			Metadata:  metadata,
			RequestID: bus.NewRequestID(),
		})
	}
}
