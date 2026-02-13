package commands

import (
	"testing"

	"github.com/MEKXH/golem/internal/config"
)

func TestChannelsSetEnabled_Telegram(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runChannelsSetEnabled("telegram", true); err != nil {
		t.Fatalf("enable telegram: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Channels.Telegram.Enabled {
		t.Fatal("expected telegram enabled=true")
	}

	if err := runChannelsSetEnabled("telegram", false); err != nil {
		t.Fatalf("disable telegram: %v", err)
	}
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Channels.Telegram.Enabled {
		t.Fatal("expected telegram enabled=false")
	}
}

func TestChannelsSetEnabled_UnknownChannel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runChannelsSetEnabled("unknown", true); err == nil {
		t.Fatal("expected error for unknown channel")
	}
}
