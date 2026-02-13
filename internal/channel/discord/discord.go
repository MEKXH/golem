package discord

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
	"github.com/bwmarrin/discordgo"
)

// Channel implements Discord bot channel.
type Channel struct {
	channel.BaseChannel
	cfg     *config.DiscordConfig
	session *discordgo.Session
	mu      sync.RWMutex
	running bool
}

// New creates a Discord channel.
func New(cfg *config.DiscordConfig, msgBus *bus.MessageBus) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	return &Channel{
		BaseChannel: channel.BaseChannel{Bus: msgBus, AllowList: allowList},
		cfg:         cfg,
	}
}

func (c *Channel) Name() string { return "discord" }

func (c *Channel) Start(ctx context.Context) error {
	if c.cfg == nil {
		return fmt.Errorf("missing discord config")
	}
	if strings.TrimSpace(c.cfg.Token) == "" {
		return fmt.Errorf("discord token is empty")
	}

	s, err := discordgo.New("Bot " + strings.TrimSpace(c.cfg.Token))
	if err != nil {
		return fmt.Errorf("create discord session: %w", err)
	}
	s.AddHandler(c.handleMessage)

	if err := s.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	c.mu.Lock()
	c.session = s
	c.running = true
	c.mu.Unlock()

	if me, err := s.User("@me"); err == nil {
		slog.Info("discord bot connected", "username", me.Username, "id", me.ID)
	}
	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	s := c.session
	c.session = nil
	c.running = false
	c.mu.Unlock()
	if s != nil {
		_ = s.Close()
	}
	return nil
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	c.mu.RLock()
	s := c.session
	running := c.running
	c.mu.RUnlock()
	if !running || s == nil {
		return fmt.Errorf("discord channel not running")
	}
	if strings.TrimSpace(msg.ChatID) == "" {
		return fmt.Errorf("discord chat id is empty")
	}

	done := make(chan error, 1)
	go func() {
		_, err := s.ChannelMessageSend(msg.ChatID, msg.Content)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("send discord message: %w", err)
		}
		return nil
	}
}

func (c *Channel) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Author == nil {
		return
	}
	if s != nil && s.State != nil && s.State.User != nil && m.Author.ID == s.State.User.ID {
		return
	}

	senderID := m.Author.ID
	senderCompound := senderID
	if m.Author.Username != "" {
		senderCompound = senderID + "|" + m.Author.Username
	}
	if !c.IsAllowed(senderCompound) {
		return
	}

	content := strings.TrimSpace(m.Content)
	media := make([]string, 0, len(m.Attachments))
	for _, att := range m.Attachments {
		if att == nil || att.URL == "" {
			continue
		}
		media = append(media, att.URL)
		if content != "" {
			content += "\n"
		}
		content += fmt.Sprintf("[attachment: %s]", att.URL)
	}
	if content == "" {
		return
	}

	metadata := map[string]any{
		"message_id": m.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	}

	c.PublishInbound(&bus.InboundMessage{
		Channel:   c.Name(),
		SenderID:  senderID,
		ChatID:    m.ChannelID,
		Content:   content,
		Timestamp: time.Now(),
		Media:     media,
		Metadata:  metadata,
		RequestID: bus.NewRequestID(),
	})
}
