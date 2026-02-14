package discord

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/voice"
	"github.com/bwmarrin/discordgo"
)

const (
	defaultTranscriptionTimeout = 30 * time.Second
	maxAudioBytes               = 25 * 1024 * 1024
)

// Channel implements Discord bot channel.
type Channel struct {
	channel.BaseChannel
	cfg                  *config.DiscordConfig
	session              *discordgo.Session
	transcriber          voice.Transcriber
	downloadAudio        func(ctx context.Context, url, fileName, mimeType string) (voice.Input, error)
	httpClient           *http.Client
	transcriptionTimeout time.Duration
	mu                   sync.RWMutex
	running              bool
}

// New creates a Discord channel.
func New(cfg *config.DiscordConfig, msgBus *bus.MessageBus, transcriber voice.Transcriber) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	ch := &Channel{
		BaseChannel:          channel.BaseChannel{Bus: msgBus, AllowList: allowList},
		cfg:                  cfg,
		transcriber:          transcriber,
		httpClient:           &http.Client{Timeout: 45 * time.Second},
		transcriptionTimeout: defaultTranscriptionTimeout,
	}
	ch.downloadAudio = ch.downloadDiscordAudio
	return ch
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
	transcribedCount := 0
	for _, att := range m.Attachments {
		if att == nil || att.URL == "" {
			continue
		}
		media = append(media, att.URL)

		if isAudioAttachment(att.ContentType, att.Filename) {
			text, err := c.tryTranscribeAttachment(context.Background(), att)
			if err != nil {
				slog.Warn("discord transcription failed", "error", err, "channel_id", m.ChannelID, "message_id", m.ID)
			}
			if strings.TrimSpace(text) != "" {
				content = appendLine(content, "[voice] "+strings.TrimSpace(text))
				transcribedCount++
				continue
			}
			label := strings.TrimSpace(att.Filename)
			if label == "" {
				label = "audio"
			}
			content = appendLine(content, fmt.Sprintf("[audio: %s]", label))
			continue
		}
		content = appendLine(content, fmt.Sprintf("[attachment: %s]", att.URL))
	}

	metadata := map[string]any{
		"message_id": m.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	}
	if transcribedCount > 0 {
		metadata["transcribed_audio"] = true
		metadata["transcribed_audio_count"] = transcribedCount
	}
	if content == "" {
		return
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

func (c *Channel) tryTranscribeAttachment(ctx context.Context, att *discordgo.MessageAttachment) (string, error) {
	if c.transcriber == nil || c.downloadAudio == nil || att == nil {
		return "", nil
	}
	if !isAudioAttachment(att.ContentType, att.Filename) {
		return "", nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	tctx, cancel := context.WithTimeout(ctx, c.transcriptionTimeout)
	defer cancel()

	input, err := c.downloadAudio(tctx, att.URL, att.Filename, att.ContentType)
	if err != nil {
		return "", err
	}
	return c.transcriber.Transcribe(tctx, input)
}

func (c *Channel) downloadDiscordAudio(ctx context.Context, url, fileName, mimeType string) (voice.Input, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return voice.Input{}, err
	}
	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return voice.Input{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return voice.Input{}, fmt.Errorf("download discord media failed: status %d", resp.StatusCode)
	}

	data, err := readLimited(resp.Body, maxAudioBytes)
	if err != nil {
		return voice.Input{}, err
	}
	return voice.Input{
		FileName: fileName,
		MIMEType: mimeType,
		Data:     data,
	}, nil
}

func isAudioAttachment(mimeType, fileName string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(mimeType, "audio/") {
		return true
	}

	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	switch ext {
	case ".ogg", ".oga", ".mp3", ".m4a", ".wav", ".flac", ".aac", ".opus", ".webm":
		return true
	default:
		return false
	}
}

func appendLine(base, suffix string) string {
	if strings.TrimSpace(suffix) == "" {
		return base
	}
	if strings.TrimSpace(base) == "" {
		return suffix
	}
	return base + "\n" + suffix
}

func readLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("audio file is too large: %d bytes (max %d)", len(data), maxBytes)
	}
	return data, nil
}
