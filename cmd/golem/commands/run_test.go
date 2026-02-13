package commands

import (
	"context"
	"testing"

	"github.com/MEKXH/golem/internal/agent"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/config"
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
	registerEnabledChannels(cfg, msgBus, mgr)

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
	registerEnabledChannels(cfg, msgBus, mgr)

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
	registerEnabledChannels(cfg, msgBus, mgr)

	if got := len(mgr.Names()); got != 0 {
		t.Fatalf("expected no channels registered, got %d", got)
	}
}
