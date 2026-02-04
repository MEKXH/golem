package commands

import (
    "context"
    "testing"

    "github.com/MEKXH/golem/internal/agent"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/channel"
    "github.com/MEKXH/golem/internal/channel/telegram"
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
    if cfg.Channels.Telegram.Enabled {
        tg := telegram.New(&cfg.Channels.Telegram, msgBus)
        mgr.Register(tg)
    }

    if len(mgr.Names()) != 0 {
        t.Fatalf("expected no channels registered")
    }
}
