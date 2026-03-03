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

// Manager 统一协调管理所有消息通道及其出站发送策略。
type Manager struct {
	channels      map[string]Channel      // 已注册的所有通道实例
	bus           *bus.MessageBus         // 消息总线，用于收发出站消息
	sendSem       chan struct{}           // 发送信号量，控制出站消息的最大并发发送数
	runtimeMetric *metrics.RuntimeMetrics // 运行时指标收集器，用于监控发送状况
	policy        DeliveryPolicy          // 出站投递策略（重试、退避、去重等）
	dedupMu       sync.Mutex
	dedupSeenAt   map[string]time.Time // 消息去重记录，防止重复发送
	rateMu        sync.Mutex
	lastSendAt    time.Time // 记录上次消息发送时间，用于速率限制
	mu            sync.RWMutex
}

const defaultMaxConcurrentSends = 16

// DeliveryPolicy 定义了出站消息的投递规则，包括重试次数、退避策略、速率限制和去重窗口。
type DeliveryPolicy struct {
	MaxConcurrentSends int           // 最大并发发送连接数
	RetryMaxAttempts   int           // 发送失败后的最大重试次数
	RetryBaseBackoff   time.Duration // 基础重试退避间隔
	RetryMaxBackoff    time.Duration // 最大重试退避间隔
	RateLimitPerSecond int           // 每秒允许发送的消息数上限
	DedupWindow        time.Duration // 消息去重的时间窗口大小
}

// NewManager 创建一个使用推荐默认策略的消息通道管理器。
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

// NewManagerWithLimit 创建一个可控制最大并发发送数的管理器。
func NewManagerWithLimit(msgBus *bus.MessageBus, maxConcurrentSends int) *Manager {
	mgr := NewManager(msgBus)
	mgr.policy.MaxConcurrentSends = maxConcurrentSends
	mgr.normalizePolicy()
	mgr.sendSem = make(chan struct{}, mgr.policy.MaxConcurrentSends)
	return mgr
}

// NewManagerWithPolicy 使用自定义的出站消息投递策略创建一个管理器。
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

// Register 将一个通道实现注册到管理器中。
func (m *Manager) Register(ch Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.Name()] = ch
}

// SetRuntimeMetrics 附加一个运行时指标收集器，用于跟踪出站消息的统计数据。
func (m *Manager) SetRuntimeMetrics(recorder *metrics.RuntimeMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runtimeMetric = recorder
}

// Names 返回所有已注册通道的名称列表。
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

// StartAll 启动管理器下属的所有消息通道，使其开始接收外部平台的入站消息。
func (m *Manager) StartAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, ch := range m.channels {
		go func(n string, c Channel) {
			slog.Info("正在启动消息通道", "name", n)
			if err := c.Start(ctx); err != nil {
				slog.Error("消息通道运行出错", "name", n, "error", err)
			}
		}(name, ch)
	}
}

// RouteOutbound 持续监控消息总线的出站队列，并将消息分发到对应的通道进行发送。
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
			// 跳过内部系统消息（如心跳消息），不路由到外部聊天平台。
			if msg.Metadata != nil {
				if mt, ok := msg.Metadata["type"]; ok && mt == "heartbeat" {
					continue
				}
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
						slog.Error("消息发送失败", "request_id", outbound.RequestID, "channel", outbound.Channel, "chat_id", outbound.ChatID, "error", err)
					}
				}(ch, msg, recorder)
			case <-ctx.Done():
				return
			}
		}
	}
}

// StopAll 停止管理器下属的所有消息通道，关闭外部平台连接。
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
