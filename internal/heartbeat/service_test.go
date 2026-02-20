package heartbeat

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/state"
)

func TestRunOnce_DispatchesToLatestActiveSession(t *testing.T) {
	var dispatched atomic.Bool
	var gotChannel, gotChatID, gotContent, gotRequestID string

	svc := NewService(
		Config{
			Enabled:  true,
			Interval: time.Minute,
			MaxIdle:  time.Hour,
		},
		func(ctx context.Context) (string, error) {
			return "cron_running=true", nil
		},
		func(ctx context.Context, channel, chatID, content, requestID string) error {
			dispatched.Store(true)
			gotChannel = channel
			gotChatID = chatID
			gotContent = content
			gotRequestID = requestID
			return nil
		},
		nil,
	)

	svc.TrackActivity("telegram", "123456")
	if err := svc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}

	if !dispatched.Load() {
		t.Fatal("expected dispatch to be called")
	}
	if gotChannel != "telegram" {
		t.Fatalf("expected channel=telegram, got %q", gotChannel)
	}
	if gotChatID != "123456" {
		t.Fatalf("expected chat_id=123456, got %q", gotChatID)
	}
	if gotRequestID == "" {
		t.Fatal("expected non-empty request_id")
	}
	if !strings.Contains(gotContent, "cron_running=true") {
		t.Fatalf("expected heartbeat content to include probe summary, got %q", gotContent)
	}
}

func TestRunOnce_SkipsWhenNoActiveSession(t *testing.T) {
	var dispatched atomic.Bool

	svc := NewService(
		Config{
			Enabled:  true,
			Interval: time.Minute,
			MaxIdle:  time.Hour,
		},
		func(ctx context.Context) (string, error) {
			return "ok", nil
		},
		func(ctx context.Context, channel, chatID, content, requestID string) error {
			dispatched.Store(true)
			return nil
		},
		nil,
	)

	if err := svc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if dispatched.Load() {
		t.Fatal("dispatch should not be called without active session")
	}
}

func TestRunOnce_ProbeFailureStillDispatches(t *testing.T) {
	var gotContent string

	svc := NewService(
		Config{
			Enabled:  true,
			Interval: time.Minute,
			MaxIdle:  time.Hour,
		},
		func(ctx context.Context) (string, error) {
			return "", errors.New("probe failed")
		},
		func(ctx context.Context, channel, chatID, content, requestID string) error {
			gotContent = content
			return nil
		},
		nil,
	)

	svc.TrackActivity("telegram", "123456")
	if err := svc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if !strings.Contains(strings.ToLower(gotContent), "degraded") {
		t.Fatalf("expected degraded heartbeat message, got %q", gotContent)
	}
}

func TestStartStop_RunsPeriodically(t *testing.T) {
	var callCount atomic.Int32

	svc := NewService(
		Config{
			Enabled:  true,
			Interval: 20 * time.Millisecond,
			MaxIdle:  time.Hour,
		},
		func(ctx context.Context) (string, error) {
			return "ok", nil
		},
		func(ctx context.Context, channel, chatID, content, requestID string) error {
			callCount.Add(1)
			return nil
		},
		nil,
	)
	svc.TrackActivity("telegram", "123456")

	if err := svc.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	defer svc.Stop()

	deadline := time.Now().Add(600 * time.Millisecond)
	for callCount.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if callCount.Load() == 0 {
		t.Fatal("expected periodic heartbeat dispatches")
	}
}

func TestRunOnce_UsesPersistedActiveSession(t *testing.T) {
	baseDir := t.TempDir()
	stateMgr := state.NewManager(baseDir)
	if err := stateMgr.SaveHeartbeatState(state.HeartbeatState{
		LastChannel: "telegram",
		LastChatID:  "persisted-chat",
		SeenAt:      time.Now(),
	}); err != nil {
		t.Fatalf("SaveHeartbeatState error: %v", err)
	}

	var dispatched atomic.Bool
	var gotChannel, gotChatID string

	svc := NewService(
		Config{
			Enabled:  true,
			Interval: time.Minute,
			MaxIdle:  time.Hour,
		},
		func(ctx context.Context) (string, error) {
			return "ok", nil
		},
		func(ctx context.Context, channel, chatID, content, requestID string) error {
			dispatched.Store(true)
			gotChannel = channel
			gotChatID = chatID
			return nil
		},
		stateMgr,
	)

	if err := svc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if !dispatched.Load() {
		t.Fatal("expected dispatch to use persisted active session")
	}
	if gotChannel != "telegram" || gotChatID != "persisted-chat" {
		t.Fatalf("unexpected persisted route channel=%q chat_id=%q", gotChannel, gotChatID)
	}
}

func TestTrackActivity_PersistsState(t *testing.T) {
	baseDir := t.TempDir()
	stateMgr := state.NewManager(baseDir)

	svc := NewService(
		Config{
			Enabled:  true,
			Interval: time.Minute,
			MaxIdle:  time.Hour,
		},
		nil,
		nil,
		stateMgr,
	)

	svc.TrackActivity("telegram", "chat-77")

	got, err := stateMgr.LoadHeartbeatState()
	if err != nil {
		t.Fatalf("LoadHeartbeatState error: %v", err)
	}
	if got.LastChannel != "telegram" || got.LastChatID != "chat-77" {
		t.Fatalf("unexpected persisted state: %+v", got)
	}
	if got.SeenAt.IsZero() {
		t.Fatal("expected non-zero seen_at")
	}
}
