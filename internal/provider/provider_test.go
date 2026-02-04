package provider

import (
    "testing"

    "github.com/MEKXH/golem/internal/config"
)

func TestNewChatModel_NoProvider(t *testing.T) {
    cfg := config.DefaultConfig()

    _, err := NewChatModel(nil, cfg)
    if err == nil {
        t.Error("expected error when no provider configured")
    }
}
