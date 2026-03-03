// Package heartbeat 实现定期心跳服务，用于监控系统状态并向用户发送活跃提醒。
package heartbeat

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	appstate "github.com/MEKXH/golem/internal/state"
)

const (
	defaultInterval = 30 * time.Minute // 默认心跳间隔
	defaultMaxIdle  = 12 * time.Hour   // 默认最大空闲时间
)

// ProbeFunc 定义了心跳探测函数的原型，返回简短的状态摘要。
type ProbeFunc func(ctx context.Context) (string, error)

// DispatchFunc 定义了将心跳内容发送到目标会话的函数原型。
type DispatchFunc func(ctx context.Context, channel, chatID, content, requestID string) error

// Config 控制心跳服务的运行时行为。
type Config struct {
	Enabled  bool          // 是否启用心跳
	Interval time.Duration // 心跳探测间隔
	MaxIdle  time.Duration // 会话最大允许空闲时间（超过后停止发送心跳）
}

type activeSession struct {
	channel string
	chatID  string
	seenAt  time.Time
}

// Service 负责定期执行健康探测，并将结果分发到最近活跃的会话中。
type Service struct {
	cfg      Config
	probe    ProbeFunc         // 健康探测逻辑
	dispatch DispatchFunc      // 消息分发逻辑
	state    *appstate.Manager // 状态管理器，用于持久化活跃信息

	now func() time.Time

	mu      sync.RWMutex
	active  activeSession
	stopCh  chan struct{}
	stopped chan struct{}
	running bool
}

// NewService 创建并初始化一个新的心跳服务。
func NewService(cfg Config, probe ProbeFunc, dispatch DispatchFunc, stateMgr *appstate.Manager) *Service {
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	if cfg.MaxIdle <= 0 {
		cfg.MaxIdle = defaultMaxIdle
	}
	svc := &Service{
		cfg:      cfg,
		probe:    probe,
		dispatch: dispatch,
		state:    stateMgr,
		now:      time.Now,
	}
	// 从持久化存储中恢复上一次的活跃会话信息
	svc.hydratePersistedState()
	return svc
}

// TrackActivity 更新并持久化最近活跃的通道与聊天 ID，用于确定心跳发送目标。
func (s *Service) TrackActivity(channel, chatID string) {
	channel = strings.TrimSpace(channel)
	chatID = strings.TrimSpace(chatID)
	if channel == "" || chatID == "" {
		return
	}
	seenAt := s.now()

	s.mu.Lock()
	s.active = activeSession{
		channel: channel,
		chatID:  chatID,
		seenAt:  seenAt,
	}
	stateMgr := s.state
	s.mu.Unlock()

	if stateMgr != nil {
		if err := stateMgr.SaveHeartbeatState(appstate.HeartbeatState{
			LastChannel: channel,
			LastChatID:  chatID,
			SeenAt:      seenAt,
		}); err != nil {
			slog.Warn("failed to persist heartbeat state", "error", err)
		}
	}
}

// IsRunning 返回服务循环当前是否处于活跃状态。
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Start 启动心跳服务的周期性循环。
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}
	if !s.cfg.Enabled {
		slog.Info("heartbeat disabled")
		return nil
	}

	s.stopCh = make(chan struct{})
	s.stopped = make(chan struct{})
	s.running = true

	go s.loop(s.stopCh, s.stopped)
	slog.Info("heartbeat service started", "interval", s.cfg.Interval.String(), "max_idle", s.cfg.MaxIdle.String())
	return nil
}

// Stop 停止心跳服务的周期性循环。
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	stopCh := s.stopCh
	stopped := s.stopped
	s.running = false
	s.stopCh = nil
	s.stopped = nil
	s.mu.Unlock()

	close(stopCh)
	<-stopped
	slog.Info("heartbeat service stopped")
}

func (s *Service) loop(stopCh <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)

	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			if err := s.RunOnce(context.Background()); err != nil {
				slog.Warn("heartbeat run failed", "error", err)
			}
		}
	}
}

// RunOnce 执行单次心跳探测并分发结果。
func (s *Service) RunOnce(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}

	target, ok := s.latestActive()
	if !ok {
		return nil
	}

	// 检查会话是否已长时间处于非活跃状态
	if s.cfg.MaxIdle > 0 && s.now().Sub(target.seenAt) > s.cfg.MaxIdle {
		slog.Debug("heartbeat skipped stale session", "channel", target.channel, "chat_id", target.chatID, "max_idle", s.cfg.MaxIdle.String())
		return nil
	}

	status := "ok"
	summary := "heartbeat check passed"

	if s.probe != nil {
		probeSummary, err := s.probe(ctx)
		if err != nil {
			status = "degraded"
			summary = err.Error()
		} else if strings.TrimSpace(probeSummary) != "" {
			summary = strings.TrimSpace(probeSummary)
		}
	}

	requestID := bus.NewRequestID()
	content := fmt.Sprintf("[heartbeat] status=%s summary=%s", status, summary)
	if s.dispatch == nil {
		slog.Debug("heartbeat dispatch skipped; dispatch func is nil", "request_id", requestID, "channel", target.channel, "chat_id", target.chatID)
		return nil
	}

	if err := s.dispatch(ctx, target.channel, target.chatID, content, requestID); err != nil {
		return err
	}

	slog.Info("heartbeat dispatched",
		"request_id", requestID,
		"channel", target.channel,
		"chat_id", target.chatID,
		"heartbeat_status", status,
	)
	return nil
}

func (s *Service) latestActive() (activeSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.active.channel == "" || s.active.chatID == "" {
		return activeSession{}, false
	}
	return s.active, true
}

func (s *Service) hydratePersistedState() {
	if s.state == nil {
		return
	}

	st, err := s.state.LoadHeartbeatState()
	if err != nil {
		slog.Warn("failed to load heartbeat state", "error", err)
		return
	}
	if st.LastChannel == "" || st.LastChatID == "" {
		return
	}

	seenAt := st.SeenAt
	if seenAt.IsZero() {
		seenAt = s.now()
	}

	s.mu.Lock()
	s.active = activeSession{
		channel: st.LastChannel,
		chatID:  st.LastChatID,
		seenAt:  seenAt,
	}
	s.mu.Unlock()
}
