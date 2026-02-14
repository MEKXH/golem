package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_SaveAndLoadHeartbeatState(t *testing.T) {
	baseDir := t.TempDir()
	mgr := NewManager(baseDir)

	wantSeenAt := time.Now().UTC().Truncate(time.Second)
	err := mgr.SaveHeartbeatState(HeartbeatState{
		LastChannel: "telegram",
		LastChatID:  "chat-42",
		SeenAt:      wantSeenAt,
	})
	if err != nil {
		t.Fatalf("SaveHeartbeatState error: %v", err)
	}

	got, err := mgr.LoadHeartbeatState()
	if err != nil {
		t.Fatalf("LoadHeartbeatState error: %v", err)
	}
	if got.LastChannel != "telegram" {
		t.Fatalf("expected last_channel=telegram, got %q", got.LastChannel)
	}
	if got.LastChatID != "chat-42" {
		t.Fatalf("expected last_chat_id=chat-42, got %q", got.LastChatID)
	}
	if !got.SeenAt.Equal(wantSeenAt) {
		t.Fatalf("expected seen_at=%s, got %s", wantSeenAt, got.SeenAt)
	}
}

func TestManager_LoadHeartbeatState_MissingFileReturnsEmpty(t *testing.T) {
	mgr := NewManager(t.TempDir())

	got, err := mgr.LoadHeartbeatState()
	if err != nil {
		t.Fatalf("LoadHeartbeatState error: %v", err)
	}
	if got.LastChannel != "" || got.LastChatID != "" {
		t.Fatalf("expected empty state, got %+v", got)
	}
}

func TestManager_LoadHeartbeatState_CorruptFileReturnsEmpty(t *testing.T) {
	baseDir := t.TempDir()
	mgr := NewManager(baseDir)

	stateFile := filepath.Join(baseDir, "state", "heartbeat.json")
	if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := os.WriteFile(stateFile, []byte("{broken"), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	got, err := mgr.LoadHeartbeatState()
	if err != nil {
		t.Fatalf("LoadHeartbeatState error: %v", err)
	}
	if got.LastChannel != "" || got.LastChatID != "" {
		t.Fatalf("expected empty state on corrupt file, got %+v", got)
	}
}
