package commands

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/agent"
	"github.com/MEKXH/golem/internal/auth"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
	"github.com/MEKXH/golem/internal/provider"
)

func TestRunCommand_WiresComponents(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = false

	msgBus := bus.NewMessageBus(10)
	model, _ := provider.NewChatModel(context.Background(), cfg)
	loop, err := agent.NewLoop(cfg, msgBus, model)
	if err != nil {
		t.Fatalf("NewLoop error: %v", err)
	}
	_ = loop.RegisterDefaultTools(cfg)

	mgr := channel.NewManager(msgBus)
	registerEnabledChannels(cfg, msgBus, mgr, nil)

	if len(mgr.Names()) != 0 {
		t.Fatalf("expected no channels registered")
	}
}

func TestRegisterEnabledChannels_RegistersAllReadyChannels(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "token"
	cfg.Channels.WhatsApp.Enabled = true
	cfg.Channels.WhatsApp.BridgeURL = "ws://127.0.0.1:8080/ws"
	cfg.Channels.Feishu.Enabled = true
	cfg.Channels.Feishu.AppID = "app-id"
	cfg.Channels.Feishu.AppSecret = "app-secret"
	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Discord.Token = "discord-token"
	cfg.Channels.Slack.Enabled = true
	cfg.Channels.Slack.BotToken = "xoxb-token"
	cfg.Channels.Slack.AppToken = "xapp-token"
	cfg.Channels.QQ.Enabled = true
	cfg.Channels.QQ.AppID = "qq-app-id"
	cfg.Channels.QQ.AppSecret = "qq-app-secret"
	cfg.Channels.DingTalk.Enabled = true
	cfg.Channels.DingTalk.ClientID = "ding-client-id"
	cfg.Channels.DingTalk.ClientSecret = "ding-client-secret"
	cfg.Channels.MaixCam.Enabled = true
	cfg.Channels.MaixCam.Host = "127.0.0.1"
	cfg.Channels.MaixCam.Port = 9000

	msgBus := bus.NewMessageBus(10)
	mgr := channel.NewManager(msgBus)
	registerEnabledChannels(cfg, msgBus, mgr, nil)

	if got := len(mgr.Names()); got != 8 {
		t.Fatalf("expected 8 registered channels, got %d", got)
	}
}

func TestRegisterEnabledChannels_SkipsNotReadyChannels(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.WhatsApp.Enabled = true
	cfg.Channels.Feishu.Enabled = true
	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Slack.Enabled = true
	cfg.Channels.QQ.Enabled = true
	cfg.Channels.DingTalk.Enabled = true
	cfg.Channels.MaixCam.Enabled = true
	cfg.Channels.MaixCam.Host = ""
	cfg.Channels.MaixCam.Port = 0

	msgBus := bus.NewMessageBus(10)
	mgr := channel.NewManager(msgBus)
	registerEnabledChannels(cfg, msgBus, mgr, nil)

	if got := len(mgr.Names()); got != 0 {
		t.Fatalf("expected no channels registered, got %d", got)
	}
}

func TestBuildHeartbeatService_RunOncePublishesOutbound(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Heartbeat.Enabled = true
	cfg.Heartbeat.Interval = 5
	cfg.Heartbeat.MaxIdleMinutes = 60

	msgBus := bus.NewMessageBus(10)
	cronSvc := cron.NewService(filepath.Join(t.TempDir(), "jobs.json"), nil)
	svc := buildHeartbeatService(cfg, msgBus, cronSvc, nil)
	if svc == nil {
		t.Fatal("expected heartbeat service")
	}

	svc.TrackActivity("telegram", "chat-1")
	if err := svc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}

	select {
	case out := <-msgBus.Outbound():
		if out.Channel != "telegram" || out.ChatID != "chat-1" {
			t.Fatalf("unexpected outbound route: %+v", out)
		}
		if out.RequestID == "" {
			t.Fatal("expected outbound request_id")
		}
		if !strings.Contains(out.Content, "[heartbeat]") {
			t.Fatalf("unexpected heartbeat content: %s", out.Content)
		}
	default:
		t.Fatal("expected heartbeat outbound message")
	}
}

func TestBuildVoiceTranscriber_DisabledReturnsNil(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Voice.Enabled = false

	if got := buildVoiceTranscriber(cfg); got != nil {
		t.Fatal("expected nil transcriber when disabled")
	}
}

func TestBuildVoiceTranscriber_OpenAIEnabledWithAPIKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Voice.Enabled = true
	cfg.Tools.Voice.Provider = "openai"
	cfg.Providers.OpenAI.APIKey = "test-key"

	if got := buildVoiceTranscriber(cfg); got == nil {
		t.Fatal("expected non-nil transcriber")
	}
}

func TestBuildVoiceTranscriber_UsesAuthStoreWhenAPIKeyMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := auth.SetCredential("openai", &auth.Credential{
		AccessToken: "auth-token",
		Provider:    "openai",
		AuthMethod:  "token",
	}); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Tools.Voice.Enabled = true
	cfg.Tools.Voice.Provider = "openai"
	cfg.Providers.OpenAI.APIKey = ""

	if got := buildVoiceTranscriber(cfg); got == nil {
		t.Fatal("expected non-nil transcriber from auth store token")
	}
}
