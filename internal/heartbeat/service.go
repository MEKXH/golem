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
	defaultInterval = 30 * time.Minute
	defaultMaxIdle  = 12 * time.Hour
)

// ProbeFunc runs a heartbeat probe and returns a short summary.
type ProbeFunc func(ctx context.Context) (string, error)

// DispatchFunc sends heartbeat output to a target session.
type DispatchFunc func(ctx context.Context, channel, chatID, content, requestID string) error

// Config controls heartbeat runtime behavior.
type Config struct {
	Enabled  bool
	Interval time.Duration
	MaxIdle  time.Duration
}

type activeSession struct {
	channel string
	chatID  string
	seenAt  time.Time
}

// Service periodically probes runtime health and sends output to the most recently active session.
type Service struct {
	cfg      Config
	probe    ProbeFunc
	dispatch DispatchFunc
	state    *appstate.Manager

	now func() time.Time

	mu      sync.RWMutex
	active  activeSession
	stopCh  chan struct{}
	stopped chan struct{}
	running bool
}

// NewService creates a heartbeat service.
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
	svc.hydratePersistedState()
	return svc
}

// TrackActivity marks a channel/chat as the newest active session for heartbeat delivery.
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

// IsRunning returns true when the service loop is active.
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Start launches the periodic heartbeat loop.
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

// Stop halts the periodic heartbeat loop.
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

// RunOnce runs a single heartbeat probe and dispatches the result to the latest active session.
func (s *Service) RunOnce(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}

	target, ok := s.latestActive()
	if !ok {
		return nil
	}

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
