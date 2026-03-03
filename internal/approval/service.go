// Package approval 实现 Golem 的人工审批流程，用于对敏感工具执行进行准入控制。
package approval

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultTTL = 15 * time.Minute // 默认审批请求有效期

// Service 负责管理审批请求的整个生命周期，包括创建、查询、决策及过期处理。
type Service struct {
	store      *Store        // 数据持久化存储
	defaultTTL time.Duration // 默认有效期
	now        func() time.Time
	mu         sync.Mutex
}

// NewService 在指定的 <workspace>/state/approvals.json 路径下创建一个审批服务。
func NewService(workspace string) *Service {
	return &Service{
		store:      NewStore(workspace),
		defaultTTL: defaultTTL,
		now:        time.Now,
	}
}

// Create 插入一个新的待审批请求。
func (s *Service) Create(input CreateInput) (Request, error) {
	toolName := strings.TrimSpace(input.ToolName)
	if toolName == "" {
		return Request{}, fmt.Errorf("tool_name is required")
	}

	argsJSON := strings.TrimSpace(input.ArgsJSON)
	reason := strings.TrimSpace(input.Reason)
	now := s.now().UTC()
	ttl := input.TTL
	if ttl <= 0 {
		ttl = s.defaultTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return Request{}, err
	}

	request := Request{
		ID:          strconv.FormatInt(data.NextID, 10),
		ToolName:    toolName,
		ArgsJSON:    argsJSON,
		Reason:      reason,
		Status:      StatusPending,
		RequestedAt: now,
		ExpiresAt:   now.Add(ttl),
	}

	data.NextID++
	data.Requests = append(data.Requests, request)

	if err := s.store.Save(data); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Approve 将一个待审批请求标记为已通过。
func (s *Service) Approve(id string, decision DecisionInput) (Request, error) {
	return s.decide(id, StatusApproved, decision, "approved")
}

// Reject 将一个待审批请求标记为已拒绝。
func (s *Service) Reject(id string, decision DecisionInput) (Request, error) {
	return s.decide(id, StatusRejected, decision, "rejected")
}

// List 根据查询条件（如 ID、状态、工具名）过滤并返回请求列表。
func (s *Service) List(query Query) ([]Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	idFilter := strings.TrimSpace(query.ID)
	statusFilter := strings.TrimSpace(string(query.Status))
	toolFilter := strings.TrimSpace(query.ToolName)

	result := make([]Request, 0, len(data.Requests))
	for _, req := range data.Requests {
		if idFilter != "" && req.ID != idFilter {
			continue
		}
		if statusFilter != "" && string(req.Status) != statusFilter {
			continue
		}
		if toolFilter != "" && !strings.EqualFold(req.ToolName, toolFilter) {
			continue
		}
		result = append(result, req)
	}
	return result, nil
}

// ExpirePending 检查并标记所有超出有效期 (TTL) 的待审批请求为已过期。
func (s *Service) ExpirePending() ([]Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	expired := make([]Request, 0)
	changed := false

	for i := range data.Requests {
		req := &data.Requests[i]
		if req.Status != StatusPending {
			continue
		}
		if req.ExpiresAt.IsZero() || req.ExpiresAt.After(now) {
			continue
		}

		req.Status = StatusExpired
		req.DecidedAt = now
		req.DecidedBy = "system"
		if strings.TrimSpace(req.DecisionNote) == "" {
			req.DecisionNote = "expired by ttl"
		}
		expired = append(expired, *req)
		changed = true
	}

	if changed {
		if err := s.store.Save(data); err != nil {
			return nil, err
		}
	}

	return expired, nil
}

func (s *Service) decide(id string, status RequestStatus, decision DecisionInput, defaultNote string) (Request, error) {
	requestID := strings.TrimSpace(id)
	if requestID == "" {
		return Request{}, fmt.Errorf("id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return Request{}, err
	}

	now := s.now().UTC()
	decidedBy := strings.TrimSpace(decision.DecidedBy)
	if decidedBy == "" {
		decidedBy = "unknown"
	}
	decisionNote := strings.TrimSpace(decision.Note)
	if decisionNote == "" {
		decisionNote = defaultNote
	}

	for i := range data.Requests {
		req := &data.Requests[i]
		if req.ID != requestID {
			continue
		}
		if req.Status != StatusPending {
			return Request{}, fmt.Errorf("request %s is not pending", requestID)
		}

		req.Status = status
		req.DecidedAt = now
		req.DecidedBy = decidedBy
		req.DecisionNote = decisionNote

		if err := s.store.Save(data); err != nil {
			return Request{}, err
		}
		return *req, nil
	}

	return Request{}, fmt.Errorf("request not found: %s", requestID)
}
