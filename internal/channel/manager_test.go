package channel

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/metrics"
)

type mockManagerChannel struct {
	BaseChannel
	name       string
	sent       atomic.Int32
	sentNotify chan struct{}
	started    bool
	stopped    bool
	sendErr    error
}

func (m *mockManagerChannel) Name() string                    { return m.name }
func (m *mockManagerChannel) Start(ctx context.Context) error { m.started = true; return nil }
func (m *mockManagerChannel) Stop(ctx context.Context) error  { m.stopped = true; return nil }
func (m *mockManagerChannel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	m.sent.Add(1)
	if m.sentNotify != nil {
		select {
		case m.sentNotify <- struct{}{}:
		default:
		}
	}
	return m.sendErr
}

func TestManager_RouteOutbound(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	mgr := NewManager(msgBus)

	ch := &mockManagerChannel{
		name:        "test",
		BaseChannel: BaseChannel{Bus: msgBus},
		sentNotify:  make(chan struct{}, 1),
	}
	mgr.Register(ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mgr.RouteOutbound(ctx)

	msgBus.PublishOutbound(&bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hi"})

	select {
	case <-ch.sentNotify:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for outbound message to be sent")
	}

	if ch.sent.Load() == 0 {
		t.Fatalf("expected message sent")
	}
}

type slowMockManagerChannel struct {
	BaseChannel
	name      string
	active    atomic.Int32
	maxActive atomic.Int32
}

func (m *slowMockManagerChannel) Name() string                    { return m.name }
func (m *slowMockManagerChannel) Start(ctx context.Context) error { return nil }
func (m *slowMockManagerChannel) Stop(ctx context.Context) error  { return nil }
func (m *slowMockManagerChannel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	current := m.active.Add(1)
	for {
		prev := m.maxActive.Load()
		if current <= prev || m.maxActive.CompareAndSwap(prev, current) {
			break
		}
	}
	time.Sleep(40 * time.Millisecond)
	m.active.Add(-1)
	return nil
}

func TestManager_RouteOutbound_LimitsConcurrency(t *testing.T) {
	msgBus := bus.NewMessageBus(20)
	mgr := NewManagerWithLimit(msgBus, 2)

	ch := &slowMockManagerChannel{name: "slow", BaseChannel: BaseChannel{Bus: msgBus}}
	mgr.Register(ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.RouteOutbound(ctx)

	for i := 0; i < 10; i++ {
		msgBus.PublishOutbound(&bus.OutboundMessage{Channel: "slow", ChatID: "1", Content: "hi"})
	}

	time.Sleep(350 * time.Millisecond)

	if got := ch.maxActive.Load(); got > 2 {
		t.Fatalf("expected max concurrent sends <= 2, got %d", got)
	}
}

func TestManager_RouteOutbound_RecordsMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	recorder := metrics.NewRuntimeMetrics(filepath.Join(tmpDir, "workspace"))

	msgBus := bus.NewMessageBus(2)
	mgr := NewManager(msgBus)
	mgr.SetRuntimeMetrics(recorder)

	ch := &mockManagerChannel{
		name:        "test",
		BaseChannel: BaseChannel{Bus: msgBus},
		sentNotify:  make(chan struct{}, 1),
	}
	mgr.Register(ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.RouteOutbound(ctx)

	msgBus.PublishOutbound(&bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hi"})

	select {
	case <-ch.sentNotify:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for outbound send")
	}

	deadline := time.After(500 * time.Millisecond)
	for {
		snap, err := metrics.ReadRuntimeSnapshot(filepath.Join(tmpDir, "workspace"))
		if err != nil {
			t.Fatalf("ReadRuntimeSnapshot() error: %v", err)
		}
		if snap.Channel.SendAttempts == 1 && snap.Channel.SendFailures == 0 {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("unexpected channel metrics snapshot: %+v", snap.Channel)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

type flakyManagerChannel struct {
	BaseChannel
	name       string
	failUntil  int32
	calls      atomic.Int32
	sentNotify chan struct{}
}

func (m *flakyManagerChannel) Name() string                    { return m.name }
func (m *flakyManagerChannel) Start(ctx context.Context) error { return nil }
func (m *flakyManagerChannel) Stop(ctx context.Context) error  { return nil }
func (m *flakyManagerChannel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	attempt := m.calls.Add(1)
	if attempt <= m.failUntil {
		return errors.New("transient send failure")
	}
	if m.sentNotify != nil {
		select {
		case m.sentNotify <- struct{}{}:
		default:
		}
	}
	return nil
}

func TestManager_RouteOutbound_RetriesRetriableChannels(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	mgr := NewManagerWithPolicy(msgBus, DeliveryPolicy{
		MaxConcurrentSends: 2,
		RetryMaxAttempts:   3,
		RetryBaseBackoff:   10 * time.Millisecond,
		RetryMaxBackoff:    20 * time.Millisecond,
		RateLimitPerSecond: 100,
		DedupWindow:        30 * time.Second,
	})

	ch := &flakyManagerChannel{
		name:        "telegram",
		BaseChannel: BaseChannel{Bus: msgBus},
		failUntil:   2,
		sentNotify:  make(chan struct{}, 1),
	}
	mgr.Register(ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.RouteOutbound(ctx)

	msgBus.PublishOutbound(&bus.OutboundMessage{
		Channel:   "telegram",
		ChatID:    "100",
		Content:   "retry please",
		RequestID: "req-retry-1",
	})

	select {
	case <-ch.sentNotify:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for retried send success")
	}

	if got := ch.calls.Load(); got != 3 {
		t.Fatalf("expected 3 send attempts (2 fail + 1 success), got %d", got)
	}
}

func TestManager_RouteOutbound_DeduplicatesRequestID(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	mgr := NewManagerWithPolicy(msgBus, DeliveryPolicy{
		MaxConcurrentSends: 2,
		RetryMaxAttempts:   1,
		RetryBaseBackoff:   5 * time.Millisecond,
		RetryMaxBackoff:    10 * time.Millisecond,
		RateLimitPerSecond: 100,
		DedupWindow:        60 * time.Second,
	})

	ch := &mockManagerChannel{
		name:        "slack",
		BaseChannel: BaseChannel{Bus: msgBus},
		sentNotify:  make(chan struct{}, 4),
	}
	mgr.Register(ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.RouteOutbound(ctx)

	outbound := &bus.OutboundMessage{
		Channel:   "slack",
		ChatID:    "C123",
		Content:   "same request",
		RequestID: "req-dup-1",
	}
	msgBus.PublishOutbound(outbound)
	msgBus.PublishOutbound(outbound)

	select {
	case <-ch.sentNotify:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for first outbound send")
	}

	time.Sleep(120 * time.Millisecond)
	if got := ch.sent.Load(); got != 1 {
		t.Fatalf("expected deduplicated sends=1, got %d", got)
	}
}

func TestManager_RouteOutbound_AppliesRateLimit(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	mgr := NewManagerWithPolicy(msgBus, DeliveryPolicy{
		MaxConcurrentSends: 1,
		RetryMaxAttempts:   1,
		RetryBaseBackoff:   5 * time.Millisecond,
		RetryMaxBackoff:    10 * time.Millisecond,
		RateLimitPerSecond: 2,
		DedupWindow:        5 * time.Second,
	})

	ch := &mockManagerChannel{
		name:        "discord",
		BaseChannel: BaseChannel{Bus: msgBus},
		sentNotify:  make(chan struct{}, 4),
	}
	mgr.Register(ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.RouteOutbound(ctx)

	start := time.Now()
	msgBus.PublishOutbound(&bus.OutboundMessage{Channel: "discord", ChatID: "1", Content: "m1", RequestID: "r1"})
	msgBus.PublishOutbound(&bus.OutboundMessage{Channel: "discord", ChatID: "1", Content: "m2", RequestID: "r2"})

	select {
	case <-ch.sentNotify:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for first send")
	}
	select {
	case <-ch.sentNotify:
	case <-time.After(1200 * time.Millisecond):
		t.Fatal("timed out waiting for second send")
	}

	elapsed := time.Since(start)
	if elapsed < 450*time.Millisecond {
		t.Fatalf("expected rate-limited spacing (~500ms), got elapsed=%s", elapsed)
	}
}

func TestManager_RouteOutbound_DedupDoesNotBlockRetryAfterFailure(t *testing.T) {
	msgBus := bus.NewMessageBus(4)
	mgr := NewManagerWithPolicy(msgBus, DeliveryPolicy{
		MaxConcurrentSends: 1,
		RetryMaxAttempts:   1,
		RetryBaseBackoff:   5 * time.Millisecond,
		RetryMaxBackoff:    10 * time.Millisecond,
		RateLimitPerSecond: 100,
		DedupWindow:        60 * time.Second,
	})

	ch := &mockManagerChannel{
		name:        "slack",
		BaseChannel: BaseChannel{Bus: msgBus},
		sentNotify:  make(chan struct{}, 4),
		sendErr:     errors.New("forced first failure"),
	}
	mgr.Register(ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.RouteOutbound(ctx)

	first := &bus.OutboundMessage{
		Channel:   "slack",
		ChatID:    "C123",
		Content:   "first should fail",
		RequestID: "req-retry-after-fail",
	}
	msgBus.PublishOutbound(first)
	select {
	case <-ch.sentNotify:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for first outbound attempt")
	}

	ch.sendErr = nil
	msgBus.PublishOutbound(first)
	select {
	case <-ch.sentNotify:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for second outbound attempt")
	}

	time.Sleep(80 * time.Millisecond)
	if got := ch.sent.Load(); got != 2 {
		t.Fatalf("expected second publish to send after first failure, got sends=%d", got)
	}
}
