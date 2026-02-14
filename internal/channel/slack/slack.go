package slack

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
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// Channel implements Slack Socket Mode channel.
type Channel struct {
	channel.BaseChannel
	cfg           *config.SlackConfig
	api           *slack.Client
	socketClient  *socketmode.Client
	botUserID     string
	transcriber   voice.Transcriber
	downloadAudio func(ctx context.Context, url, fileName, mimeType string) (voice.Input, error)
	httpClient    *http.Client

	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// New creates a Slack channel.
func New(cfg *config.SlackConfig, msgBus *bus.MessageBus, transcriber voice.Transcriber) *Channel {
	allowList := make(map[string]bool)
	for _, id := range cfg.AllowFrom {
		allowList[id] = true
	}
	ch := &Channel{
		BaseChannel: channel.BaseChannel{Bus: msgBus, AllowList: allowList},
		cfg:         cfg,
		transcriber: transcriber,
		httpClient:  &http.Client{Timeout: 45 * time.Second},
	}
	ch.downloadAudio = ch.downloadSlackAudio
	return ch
}

func (c *Channel) Name() string { return "slack" }

func (c *Channel) Start(ctx context.Context) error {
	if c.cfg == nil {
		return fmt.Errorf("missing slack config")
	}
	if strings.TrimSpace(c.cfg.BotToken) == "" || strings.TrimSpace(c.cfg.AppToken) == "" {
		return fmt.Errorf("slack bot_token and app_token are required")
	}

	api := slack.New(c.cfg.BotToken, slack.OptionAppLevelToken(c.cfg.AppToken))
	authResp, err := api.AuthTest()
	if err != nil {
		return fmt.Errorf("slack auth failed: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	socketClient := socketmode.New(api)

	c.mu.Lock()
	c.api = api
	c.socketClient = socketClient
	c.botUserID = authResp.UserID
	c.running = true
	c.ctx = runCtx
	c.cancel = cancel
	c.mu.Unlock()

	go c.eventLoop()
	go func() {
		if err := socketClient.RunContext(runCtx); err != nil && runCtx.Err() == nil {
			slog.Error("slack socket mode exited", "error", err)
		}
	}()

	slog.Info("slack channel connected", "team", authResp.Team, "bot_user_id", authResp.UserID)
	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.running = false
	c.socketClient = nil
	c.api = nil
	c.mu.Unlock()
	return nil
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	c.mu.RLock()
	api := c.api
	running := c.running
	c.mu.RUnlock()
	if !running || api == nil {
		return fmt.Errorf("slack channel not running")
	}

	channelID, threadTS := parseChatID(msg.ChatID)
	if strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("invalid slack chat id: %q", msg.ChatID)
	}

	opts := []slack.MsgOption{slack.MsgOptionText(msg.Content, false)}
	if threadTS != "" {
		opts = append(opts, slack.MsgOptionTS(threadTS))
	}

	_, _, err := api.PostMessageContext(ctx, channelID, opts...)
	if err != nil {
		return fmt.Errorf("send slack message: %w", err)
	}
	return nil
}

func (c *Channel) eventLoop() {
	for {
		c.mu.RLock()
		runCtx := c.ctx
		socketClient := c.socketClient
		c.mu.RUnlock()
		if runCtx == nil || socketClient == nil {
			return
		}

		select {
		case <-runCtx.Done():
			return
		case evt, ok := <-socketClient.Events:
			if !ok {
				return
			}
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				c.handleEventsAPI(evt)
			case socketmode.EventTypeInteractive:
				if evt.Request != nil {
					socketClient.Ack(*evt.Request)
				}
			case socketmode.EventTypeSlashCommand:
				c.handleSlashCommand(evt)
			}
		}
	}
}

func (c *Channel) handleEventsAPI(evt socketmode.Event) {
	if c.socketClient != nil && evt.Request != nil {
		c.socketClient.Ack(*evt.Request)
	}

	eventData, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		return
	}

	switch inner := eventData.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		c.handleMessageEvent(inner)
	case *slackevents.AppMentionEvent:
		c.handleMentionEvent(inner)
	}
}

func (c *Channel) handleMessageEvent(ev *slackevents.MessageEvent) {
	if ev == nil {
		return
	}
	if ev.User == "" || ev.BotID != "" || ev.SubType == "bot_message" {
		return
	}

	senderID := ev.User
	if !c.IsAllowed(senderID) {
		return
	}

	content := strings.TrimSpace(c.stripMention(ev.Text))
	media := make([]string, 0)
	var transcribed string
	for _, file := range c.extractFiles(ev) {
		url := strings.TrimSpace(file.URLPrivateDownload)
		if url == "" {
			url = strings.TrimSpace(file.URLPrivate)
		}
		if url != "" {
			media = append(media, url)
			if content != "" {
				content += "\n"
			}
			content += fmt.Sprintf("[attachment: %s]", url)
		}

		if transcribed == "" {
			if text, err := c.tryTranscribeFile(context.Background(), file); err != nil {
				slog.Warn("slack transcription failed", "error", err, "channel_id", ev.Channel, "message_ts", ev.TimeStamp)
			} else if strings.TrimSpace(text) != "" {
				transcribed = strings.TrimSpace(text)
			}
		}
	}
	if transcribed != "" {
		if content == "" {
			content = transcribed
		} else {
			content += "\n\n[voice] " + transcribed
		}
	}
	if content == "" {
		return
	}

	chatID := ev.Channel
	if ev.ThreadTimeStamp != "" {
		chatID = ev.Channel + "/" + ev.ThreadTimeStamp
	}

	metadata := map[string]any{
		"message_ts": ev.TimeStamp,
		"channel_id": ev.Channel,
		"thread_ts":  ev.ThreadTimeStamp,
	}
	if transcribed != "" {
		metadata["transcribed_audio"] = true
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

func (c *Channel) handleMentionEvent(ev *slackevents.AppMentionEvent) {
	if ev == nil {
		return
	}
	if ev.User == "" {
		return
	}
	if !c.IsAllowed(ev.User) {
		return
	}

	content := strings.TrimSpace(c.stripMention(ev.Text))
	if content == "" {
		return
	}

	chatID := ev.Channel
	if ev.ThreadTimeStamp != "" {
		chatID = ev.Channel + "/" + ev.ThreadTimeStamp
	} else if ev.TimeStamp != "" {
		chatID = ev.Channel + "/" + ev.TimeStamp
	}

	c.PublishInbound(&bus.InboundMessage{
		Channel:   c.Name(),
		SenderID:  ev.User,
		ChatID:    chatID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"message_ts": ev.TimeStamp,
			"channel_id": ev.Channel,
			"thread_ts":  ev.ThreadTimeStamp,
			"is_mention": true,
		},
		RequestID: bus.NewRequestID(),
	})
}

func (c *Channel) handleSlashCommand(evt socketmode.Event) {
	if c.socketClient != nil && evt.Request != nil {
		c.socketClient.Ack(*evt.Request)
	}
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		return
	}
	if cmd.UserID == "" {
		return
	}
	if !c.IsAllowed(cmd.UserID) {
		return
	}

	content := strings.TrimSpace(cmd.Text)
	if content == "" {
		content = "help"
	}
	c.PublishInbound(&bus.InboundMessage{
		Channel:   c.Name(),
		SenderID:  cmd.UserID,
		ChatID:    cmd.ChannelID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"is_command": true,
			"command":    cmd.Command,
			"trigger_id": cmd.TriggerID,
		},
		RequestID: bus.NewRequestID(),
	})
}

func (c *Channel) stripMention(text string) string {
	c.mu.RLock()
	botUserID := c.botUserID
	c.mu.RUnlock()
	if botUserID == "" {
		return strings.TrimSpace(text)
	}
	mention := fmt.Sprintf("<@%s>", botUserID)
	text = strings.ReplaceAll(text, mention, "")
	return strings.TrimSpace(text)
}

func parseChatID(chatID string) (channelID, threadTS string) {
	parts := strings.SplitN(chatID, "/", 2)
	channelID = parts[0]
	if len(parts) > 1 {
		threadTS = parts[1]
	}
	return
}

func (c *Channel) extractFiles(ev *slackevents.MessageEvent) []slack.File {
	if ev == nil || ev.Message == nil {
		return nil
	}
	return ev.Message.Files
}

func (c *Channel) tryTranscribeFile(ctx context.Context, file slack.File) (string, error) {
	if c.transcriber == nil || c.downloadAudio == nil {
		return "", nil
	}
	if !isAudioSlackFile(file.Mimetype, file.Name) {
		return "", nil
	}

	url := strings.TrimSpace(file.URLPrivateDownload)
	if url == "" {
		url = strings.TrimSpace(file.URLPrivate)
	}
	if url == "" {
		return "", nil
	}

	input, err := c.downloadAudio(ctx, url, file.Name, file.Mimetype)
	if err != nil {
		return "", err
	}
	return c.transcriber.Transcribe(ctx, input)
}

func (c *Channel) downloadSlackAudio(ctx context.Context, url, fileName, mimeType string) (voice.Input, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return voice.Input{}, err
	}
	token := strings.TrimSpace(c.cfg.BotToken)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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
		return voice.Input{}, fmt.Errorf("download slack media failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return voice.Input{}, err
	}
	return voice.Input{
		FileName: fileName,
		MIMEType: mimeType,
		Data:     data,
	}, nil
}

func isAudioSlackFile(mimeType, fileName string) bool {
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
