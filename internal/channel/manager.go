package channel

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/metrics"
)

// Manager coordinates all channels
type Manager struct {
	channels      map[string]Channel
	bus           *bus.MessageBus
	sendSem       chan struct{}
	runtimeMetric *metrics.RuntimeMetrics
	policy        DeliveryPolicy
	dedupMu       sync.Mutex
	dedupSeenAt   map[string]time.Time
	rateMu        sync.Mutex
	lastSendAt    time.Time
	mu            sync.RWMutex
}

const defaultMaxConcurrentSends = 16

// DeliveryPolicy defines outbound retry, backoff, rate limit and dedup behavior.
type DeliveryPolicy struct {
	MaxConcurrentSends int
	RetryMaxAttempts   int
	RetryBaseBackoff   time.Duration
	RetryMaxBackoff    time.Duration
	RateLimitPerSecond int
	DedupWindow        time.Duration
}

// NewManager creates a channel manager
func NewManager(msgBus *bus.MessageBus) *Manager {
	return NewManagerWithPolicy(msgBus, DeliveryPolicy{
		MaxConcurrentSends: defaultMaxConcurrentSends,
		RetryMaxAttempts:   3,
		RetryBaseBackoff:   200 * time.Millisecond,
		RetryMaxBackoff:    2 * time.Second,
		RateLimitPerSecond: 20,
		DedupWindow:        30 * time.Second,
	})
}

// NewManagerWithLimit creates a channel manager with bounded outbound send concurrency.
func NewManagerWithLimit(msgBus *bus.MessageBus, maxConcurrentSends int) *Manager {
	mgr := NewManager(msgBus)
	mgr.policy.MaxConcurrentSends = maxConcurrentSends
	mgr.normalizePolicy()
	mgr.sendSem = make(chan struct{}, mgr.policy.MaxConcurrentSends)
	return mgr
}

// NewManagerWithPolicy creates a manager with custom outbound delivery policy.
func NewManagerWithPolicy(msgBus *bus.MessageBus, policy DeliveryPolicy) *Manager {
	normalized := normalizeDeliveryPolicy(policy)
	return &Manager{
		channels:    make(map[string]Channel),
		bus:         msgBus,
		sendSem:     make(chan struct{}, normalized.MaxConcurrentSends),
		policy:      normalized,
		dedupSeenAt: make(map[string]time.Time),
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
			ch, recorder, ok := m.resolveChannel(msg.Channel)
			if !ok {
				continue
			}
			select {
			case m.sendSem <- struct{}{}:
				go func(c Channel, outbound *bus.OutboundMessage, metricRecorder *metrics.RuntimeMetrics) {
					defer func() { <-m.sendSem }()
					if err := m.sendWithPolicy(ctx, c, outbound, metricRecorder); err != nil {
						slog.Error("send outbound failed", "request_id", outbound.RequestID, "channel", outbound.Channel, "chat_id", outbound.ChatID, "error", err)
					}
				}(ch, msg, recorder)
			case <-ctx.Done():
				return
			}
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

func (m *Manager) resolveChannel(name string) (Channel, *metrics.RuntimeMetrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[name]
	if !ok {
		return nil, nil, false
	}
	return ch, m.runtimeMetric, true
}

func (m *Manager) sendWithPolicy(ctx context.Context, c Channel, outbound *bus.OutboundMessage, recorder *metrics.RuntimeMetrics) error {
	if outbound == nil {
		return fmt.Errorf("outbound message is nil")
	}
	dedupKey, duplicate := m.beginDedup(outbound)
	if duplicate {
		slog.Warn("skip duplicate outbound",
			"request_id", outbound.RequestID,
			"channel", outbound.Channel,
			"chat_id", outbound.ChatID,
		)
		return nil
	}
	delivered := false
	defer func() {
		if !delivered {
			m.releaseDedup(dedupKey)
		}
	}()

	attempts := m.policy.RetryMaxAttempts
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := m.waitRateLimit(ctx); err != nil {
			return err
		}
		err := c.Send(ctx, outbound)
		m.recordChannelMetric(recorder, err == nil, outbound, err, attempt, attempts)
		if err == nil {
			m.confirmDedup(dedupKey)
			delivered = true
			return nil
		}
		lastErr = err

		if !m.shouldRetry(outbound.Channel, attempt, attempts, err) {
			break
		}
		if backoffErr := m.waitBackoff(ctx, attempt); backoffErr != nil {
			return backoffErr
		}
	}

	return fmt.Errorf("final send failure after %d attempt(s): %w", attempts, lastErr)
}

func (m *Manager) shouldRetry(channelName string, attempt, maxAttempts int, err error) bool {
	if err == nil || attempt >= maxAttempts {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(channelName))
	return name == "telegram" || name == "discord" || name == "slack"
}

func (m *Manager) waitBackoff(ctx context.Context, attempt int) error {
	backoff := m.policy.RetryBaseBackoff
	if backoff <= 0 {
		return nil
	}
	if attempt > 1 {
		backoff = backoff * time.Duration(1<<(attempt-1))
	}
	if max := m.policy.RetryMaxBackoff; max > 0 && backoff > max {
		backoff = max
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (m *Manager) waitRateLimit(ctx context.Context) error {
	rate := m.policy.RateLimitPerSecond
	if rate <= 0 {
		return nil
	}
	interval := time.Second / time.Duration(rate)
	if interval <= 0 {
		return nil
	}

	m.rateMu.Lock()
	now := time.Now()
	nextAllowed := m.lastSendAt.Add(interval)
	wait := time.Duration(0)
	if now.Before(nextAllowed) {
		wait = nextAllowed.Sub(now)
		m.lastSendAt = nextAllowed
	} else {
		m.lastSendAt = now
	}
	m.rateMu.Unlock()

	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (m *Manager) dedupKey(outbound *bus.OutboundMessage) string {
	if outbound == nil {
		return ""
	}
	requestID := strings.TrimSpace(outbound.RequestID)
	if requestID == "" {
		return ""
	}
	return outbound.Channel + "|" + outbound.ChatID + "|" + requestID
}

func (m *Manager) beginDedup(outbound *bus.OutboundMessage) (string, bool) {
	key := m.dedupKey(outbound)
	if key == "" || m.policy.DedupWindow <= 0 {
		return "", false
	}

	now := time.Now()
	m.dedupMu.Lock()
	defer m.dedupMu.Unlock()
	cutoff := now.Add(-m.policy.DedupWindow)
	for k, ts := range m.dedupSeenAt {
		if ts.Before(cutoff) {
			delete(m.dedupSeenAt, k)
		}
	}
	if ts, ok := m.dedupSeenAt[key]; ok && now.Sub(ts) <= m.policy.DedupWindow {
		return key, true
	}
	m.dedupSeenAt[key] = now
	return key, false
}

func (m *Manager) confirmDedup(key string) {
	if key == "" || m.policy.DedupWindow <= 0 {
		return
	}
	m.dedupMu.Lock()
	m.dedupSeenAt[key] = time.Now()
	m.dedupMu.Unlock()
}

func (m *Manager) releaseDedup(key string) {
	if key == "" || m.policy.DedupWindow <= 0 {
		return
	}
	m.dedupMu.Lock()
	delete(m.dedupSeenAt, key)
	m.dedupMu.Unlock()
}

func (m *Manager) recordChannelMetric(recorder *metrics.RuntimeMetrics, success bool, outbound *bus.OutboundMessage, sendErr error, attempt, maxAttempts int) {
	if recorder == nil {
		return
	}
	snapshot, recordErr := recorder.RecordChannelSend(success)
	if recordErr != nil {
		slog.Warn("record runtime metrics failed", "scope", "channel", "error", recordErr)
		return
	}
	if success {
		return
	}

	slog.Error("send outbound attempt failed",
		"request_id", outbound.RequestID,
		"channel", outbound.Channel,
		"chat_id", outbound.ChatID,
		"attempt", attempt,
		"max_attempts", maxAttempts,
		"error", sendErr,
		"channel_send_attempts", snapshot.Channel.SendAttempts,
		"channel_send_failure_ratio", snapshot.Channel.FailureRatio(),
	)
}

func normalizeDeliveryPolicy(policy DeliveryPolicy) DeliveryPolicy {
	if policy.MaxConcurrentSends <= 0 {
		policy.MaxConcurrentSends = defaultMaxConcurrentSends
	}
	if policy.RetryMaxAttempts <= 0 {
		policy.RetryMaxAttempts = 3
	}
	if policy.RetryBaseBackoff <= 0 {
		policy.RetryBaseBackoff = 200 * time.Millisecond
	}
	if policy.RetryMaxBackoff <= 0 {
		policy.RetryMaxBackoff = 2 * time.Second
	}
	if policy.RetryMaxBackoff < policy.RetryBaseBackoff {
		policy.RetryMaxBackoff = policy.RetryBaseBackoff
	}
	if policy.RateLimitPerSecond <= 0 {
		policy.RateLimitPerSecond = 20
	}
	if policy.DedupWindow <= 0 {
		policy.DedupWindow = 30 * time.Second
	}
	return policy
}

func (m *Manager) normalizePolicy() {
	m.policy = normalizeDeliveryPolicy(m.policy)
}
