package telegram

import (
    "testing"

    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/config"
)

func TestTelegramChannel_AllowList(t *testing.T) {
    cfg := &config.TelegramConfig{
        AllowFrom: []string{"u1", "u2"},
    }

    msgBus := bus.NewMessageBus(1)
    ch := New(cfg, msgBus)

    if ch.IsAllowed("u1") != true {
        t.Fatalf("expected u1 allowed")
    }
    if ch.IsAllowed("u3") != false {
        t.Fatalf("expected u3 denied")
    }
}
