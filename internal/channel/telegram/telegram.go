package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/render"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	boldStarRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	boldUnderRe  = regexp.MustCompile(`__(.+?)__`)
	codeInlineRe = regexp.MustCompile("`([^`]+)`")
)

// Channel implements Telegram bot
type Channel struct {
	channel.BaseChannel
	cfg *config.TelegramConfig
	bot *tgbotapi.BotAPI
}

// New creates a Telegram channel
func New(cfg *config.TelegramConfig, msgBus *bus.MessageBus) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	return &Channel{
		BaseChannel: channel.BaseChannel{
			Bus:       msgBus,
			AllowList: allowList,
		},
		cfg: cfg,
	}
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
			c.handleMessage(update.Message)
		}
	}
}

func (c *Channel) handleMessage(msg *tgbotapi.Message) {
	senderID := fmt.Sprintf("%d", msg.From.ID)

	if !c.IsAllowed(senderID) {
		slog.Debug("unauthorized sender", "id", senderID)
		return
	}

	content := msg.Text
	if content == "" {
		content = msg.Caption
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
		Metadata: map[string]any{
			"message_id": msg.MessageID,
			"username":   msg.From.UserName,
		},
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
