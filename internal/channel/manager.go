package channel

import (
	"context"
	"log/slog"
	"sync"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/metrics"
)

// Manager coordinates all channels
type Manager struct {
	channels      map[string]Channel
	bus           *bus.MessageBus
	sendSem       chan struct{}
	runtimeMetric *metrics.RuntimeMetrics
	mu            sync.RWMutex
}

const defaultMaxConcurrentSends = 16

// NewManager creates a channel manager
func NewManager(msgBus *bus.MessageBus) *Manager {
	return NewManagerWithLimit(msgBus, defaultMaxConcurrentSends)
}

// NewManagerWithLimit creates a channel manager with bounded outbound send concurrency.
func NewManagerWithLimit(msgBus *bus.MessageBus, maxConcurrentSends int) *Manager {
	if maxConcurrentSends <= 0 {
		maxConcurrentSends = 1
	}
	return &Manager{
		channels: make(map[string]Channel),
		bus:      msgBus,
		sendSem:  make(chan struct{}, maxConcurrentSends),
	}
}

// Register adds a channel
func (m *Manager) Register(ch Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.Name()] = ch
}

// SetRuntimeMetrics attaches a recorder used for outbound send metrics.
func (m *Manager) SetRuntimeMetrics(recorder *metrics.RuntimeMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runtimeMetric = recorder
}

// Names returns registered channel names
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

// StartAll starts all channels
func (m *Manager) StartAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, ch := range m.channels {
		go func(n string, c Channel) {
			slog.Info("starting channel", "name", n)
			if err := c.Start(ctx); err != nil {
				slog.Error("channel error", "name", n, "error", err)
			}
		}(name, ch)
	}
}

// RouteOutbound sends outbound messages to appropriate channels
func (m *Manager) RouteOutbound(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-m.bus.Outbound():
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			m.mu.RLock()
			if ch, ok := m.channels[msg.Channel]; ok {
				metricRecorder := m.runtimeMetric
				select {
				case m.sendSem <- struct{}{}:
					go func(c Channel, outbound *bus.OutboundMessage, recorder *metrics.RuntimeMetrics) {
						defer func() { <-m.sendSem }()
						err := c.Send(ctx, outbound)

						if recorder != nil {
							snapshot, recordErr := recorder.RecordChannelSend(err == nil)
							if recordErr != nil {
								slog.Warn("record runtime metrics failed", "scope", "channel", "error", recordErr)
							} else if err != nil {
								slog.Error("send outbound failed",
									"request_id", outbound.RequestID,
									"channel", outbound.Channel,
									"chat_id", outbound.ChatID,
									"error", err,
									"channel_send_attempts", snapshot.Channel.SendAttempts,
									"channel_send_failure_ratio", snapshot.Channel.FailureRatio(),
								)
								return
							}
						}

						if err != nil {
							slog.Error("send outbound failed", "request_id", outbound.RequestID, "channel", outbound.Channel, "chat_id", outbound.ChatID, "error", err)
						}
					}(ch, msg, metricRecorder)
				case <-ctx.Done():
					m.mu.RUnlock()
					return
				}
			}
			m.mu.RUnlock()
		}
	}
}

// StopAll stops all channels
func (m *Manager) StopAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ch := range m.channels {
		_ = ch.Stop(ctx)
	}
}
