package commands

import (
	"strings"
	"testing"
)

func TestChannelsSetEnabled_Telegram(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runChannelsSetEnabled("telegram", true); err != nil {
		t.Fatalf("enable telegram: %v", err)
	}
	if err := runChannelsSetEnabled("telegram", false); err != nil {
		t.Fatalf("disable telegram: %v", err)
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

func TestChannelsSetEnabled_AllSupportedChannels(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	channels := []string{
		"telegram",
		"whatsapp",
		"feishu",
		"discord",
		"slack",
		"qq",
		"dingtalk",
		"maixcam",
	}

	for _, name := range channels {
		if err := runChannelsSetEnabled(name, true); err != nil {
			t.Fatalf("enable %s: %v", name, err)
		}
		if err := runChannelsSetEnabled(name, false); err != nil {
			t.Fatalf("disable %s: %v", name, err)
		}
	}
}

func TestChannelsList_ContainsAllSupportedChannels(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	out := captureOutput(t, func() {
		if err := runChannelsList(nil, nil); err != nil {
			t.Fatalf("runChannelsList: %v", err)
		}
	})

	for _, name := range []string{"telegram", "whatsapp", "feishu", "discord", "slack", "qq", "dingtalk", "maixcam"} {
		if !strings.Contains(out, name) {
			t.Fatalf("expected channel %q in output, got: %s", name, out)
		}
	}
}
