package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/render"
	"github.com/MEKXH/golem/internal/voice"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	boldStarRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	boldUnderRe  = regexp.MustCompile(`__(.+?)__`)
	codeInlineRe = regexp.MustCompile("`([^`]+)`")
)

const (
	defaultTranscriptionTimeout = 30 * time.Second
	maxAudioBytes               = 25 * 1024 * 1024
)

// Channel implements Telegram bot
type Channel struct {
	channel.BaseChannel
	cfg                  *config.TelegramConfig
	bot                  *tgbotapi.BotAPI
	transcriber          voice.Transcriber
	downloadVoice        func(ctx context.Context, fileID, fileName, mimeType string) (voice.Input, error)
	httpClient           *http.Client
	transcriptionTimeout time.Duration
}

// New creates a Telegram channel
func New(cfg *config.TelegramConfig, msgBus *bus.MessageBus, transcriber voice.Transcriber) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	ch := &Channel{
		BaseChannel: channel.BaseChannel{
			Bus:       msgBus,
			AllowList: allowList,
		},
		cfg:                  cfg,
		transcriber:          transcriber,
		httpClient:           &http.Client{Timeout: 45 * time.Second},
		transcriptionTimeout: defaultTranscriptionTimeout,
	}
	ch.downloadVoice = ch.downloadTelegramVoice
	return ch
}

func (c *Channel) Name() string { return "telegram" }

func (c *Channel) Start(ctx context.Context) error {
	bot, err := tgbotapi.NewBotAPI(c.cfg.Token)
	if err != nil {
		return fmt.Errorf("telegram init failed: %w", err)
	}
	c.bot = bot

	slog.Info("telegram bot connected", "username", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil {
				continue
			}
			c.handleMessage(ctx, update.Message)
		}
	}
}

func (c *Channel) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	if msg == nil || msg.From == nil || msg.Chat == nil {
		return
	}
	senderID := fmt.Sprintf("%d", msg.From.ID)

	if !c.IsAllowed(senderID) {
		slog.Debug("unauthorized sender", "id", senderID)
		return
	}

	content := msg.Text
	if content == "" {
		content = msg.Caption
	}

	metadata := map[string]any{
		"message_id": msg.MessageID,
		"username":   msg.From.UserName,
	}

	transcribed, hasAudio, err := c.tryTranscribeAudio(ctx, msg)
	if err != nil {
		slog.Warn("telegram transcription failed", "error", err, "chat_id", msg.Chat.ID, "message_id", msg.MessageID)
	}
	if transcribed != "" {
		if content == "" {
			content = transcribed
		} else {
			content = content + "\n\n[voice] " + transcribed
		}
		metadata["transcribed_audio"] = true
	} else if hasAudio && strings.TrimSpace(content) == "" {
		content = telegramAudioPlaceholder(msg)
	}
	if content == "" {
		return
	}

	c.PublishInbound(&bus.InboundMessage{
		Channel:   "telegram",
		SenderID:  senderID,
		ChatID:    fmt.Sprintf("%d", msg.Chat.ID),
		Content:   content,
		Timestamp: time.Now(),
		RequestID: bus.NewRequestID(),
		Metadata:  metadata,
	})
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	if c.bot == nil {
		return fmt.Errorf("bot not initialized")
	}

	chatID, err := parseInt64(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat id %q: %w", msg.ChatID, err)
	}
	html := renderMessageHTML(msg.Content)

	tgMsg := tgbotapi.NewMessage(chatID, html)
	tgMsg.ParseMode = "HTML"

	_, err = c.bot.Send(tgMsg)
	if err != nil {
		tgMsg.ParseMode = ""
		tgMsg.Text = msg.Content
		_, err = c.bot.Send(tgMsg)
	}
	return err
}

func (c *Channel) Stop(ctx context.Context) error {
	if c.bot != nil {
		c.bot.StopReceivingUpdates()
	}
	return nil
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func renderMessageHTML(content string) string {
	think, main, hasThink := render.SplitThink(content)
	if hasThink {
		thinkHTML := markdownToHTML(think)
		mainHTML := markdownToHTML(main)
		if mainHTML == "" {
			return "Thinking:\n" + thinkHTML
		}
		return "Thinking:\n" + thinkHTML + "\n\n" + mainHTML
	}
	return markdownToHTML(content)
}

func markdownToHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = boldStarRe.ReplaceAllString(text, "<b>$1</b>")
	text = boldUnderRe.ReplaceAllString(text, "<b>$1</b>")
	text = codeInlineRe.ReplaceAllString(text, "<code>$1</code>")
	return text
}

func (c *Channel) tryTranscribeAudio(ctx context.Context, msg *tgbotapi.Message) (string, bool, error) {
	fileID, fileName, mimeType := telegramAudioDescriptor(msg)
	if fileID == "" {
		return "", false, nil
	}

	if c.transcriber == nil || c.downloadVoice == nil {
		return "", true, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	tctx, cancel := context.WithTimeout(ctx, c.transcriptionTimeout)
	defer cancel()

	input, err := c.downloadVoice(tctx, fileID, fileName, mimeType)
	if err != nil {
		return "", true, err
	}
	text, err := c.transcriber.Transcribe(tctx, input)
	if err != nil {
		return "", true, err
	}
	return text, true, nil
}

func telegramAudioDescriptor(msg *tgbotapi.Message) (fileID, fileName, mimeType string) {
	if msg == nil {
		return "", "", ""
	}
	if msg.Voice != nil && strings.TrimSpace(msg.Voice.FileID) != "" {
		return strings.TrimSpace(msg.Voice.FileID), "voice.ogg", strings.TrimSpace(msg.Voice.MimeType)
	}
	if msg.Audio != nil && strings.TrimSpace(msg.Audio.FileID) != "" {
		name := strings.TrimSpace(msg.Audio.FileName)
		if name == "" {
			name = "audio.mp3"
		}
		return strings.TrimSpace(msg.Audio.FileID), name, strings.TrimSpace(msg.Audio.MimeType)
	}
	return "", "", ""
}

func (c *Channel) downloadTelegramVoice(ctx context.Context, fileID, fileName, mimeType string) (voice.Input, error) {
	if c.bot == nil {
		return voice.Input{}, fmt.Errorf("telegram bot not initialized")
	}
	url, err := c.bot.GetFileDirectURL(fileID)
	if err != nil {
		return voice.Input{}, err
	}

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
		return voice.Input{}, fmt.Errorf("download telegram media failed: status %d", resp.StatusCode)
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

func telegramAudioPlaceholder(msg *tgbotapi.Message) string {
	if msg != nil && msg.Audio != nil {
		return "[audio]"
	}
	return "[voice]"
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
