package channel

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/bus"
)

type mockManagerChannel struct {
	BaseChannel
	name    string
	sent    int
	started bool
	stopped bool
}

func (m *mockManagerChannel) Name() string                    { return m.name }
func (m *mockManagerChannel) Start(ctx context.Context) error { m.started = true; return nil }
func (m *mockManagerChannel) Stop(ctx context.Context) error  { m.stopped = true; return nil }
func (m *mockManagerChannel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	m.sent++
	return nil
}

func TestManager_RouteOutbound(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	mgr := NewManager(msgBus)

	ch := &mockManagerChannel{name: "test", BaseChannel: BaseChannel{Bus: msgBus}}
	mgr.Register(ch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mgr.RouteOutbound(ctx)

	msgBus.PublishOutbound(&bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hi"})

	<-time.After(10 * time.Millisecond)

	if ch.sent == 0 {
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
