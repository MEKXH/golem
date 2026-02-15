package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/approval"
	"github.com/MEKXH/golem/internal/audit"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/policy"
	"github.com/MEKXH/golem/internal/tools"
)

type runtimeGuard struct {
	baseMode        policy.Mode
	requireApproval []string
	offUntil        time.Time
	approvalService *approval.Service
	auditWriter     *audit.Writer
}

func (l *Loop) configureRuntimeGuard(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	guard := &runtimeGuard{
		baseMode:        policy.Mode(strings.TrimSpace(cfg.Policy.Mode)),
		requireApproval: append([]string(nil), cfg.Policy.RequireApproval...),
		approvalService: approval.NewService(l.workspacePath),
		auditWriter:     audit.NewWriter(l.workspacePath),
	}

	if ttlRaw := strings.TrimSpace(cfg.Policy.OffTTL); ttlRaw != "" {
		ttl, err := time.ParseDuration(ttlRaw)
		if err != nil {
			return fmt.Errorf("parse policy.off_ttl: %w", err)
		}
		guard.offUntil = l.nowUTC().Add(ttl)
	}

	l.runtimeGuard = guard
	l.tools.SetGuard(l.evaluateToolGuard)
	return nil
}

// AuditRuntimePolicyStartup writes startup policy audit events.
func (l *Loop) AuditRuntimePolicyStartup(ctx context.Context, cfg *config.Config) {
	if cfg == nil {
		return
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.Policy.Mode))
	if mode == "" {
		mode = string(policy.ModeStrict)
	}

	offTTL := strings.TrimSpace(cfg.Policy.OffTTL)
	offTTLResult := offTTL
	if offTTLResult == "" {
		offTTLResult = "none"
	}

	requireApproval := "-"
	if len(cfg.Policy.RequireApproval) > 0 {
		requireApproval = strings.Join(cfg.Policy.RequireApproval, ",")
	}

	startupResult := fmt.Sprintf("mode=%s off_ttl=%s allow_persistent_off=%t require_approval=%s",
		mode,
		offTTLResult,
		cfg.Policy.AllowPersistentOff,
		requireApproval,
	)
	l.appendAuditEvent(ctx, "policy_startup", "", "", startupResult)

	if mode == string(policy.ModeOff) && offTTL == "" {
		l.appendAuditEvent(ctx, "policy_startup_persistent_off", "", "",
			"high-risk: policy.mode=off without policy.off_ttl keeps runtime guardrails disabled indefinitely")
	}
}

func (l *Loop) evaluateToolGuard(ctx context.Context, name, argsJSON string) (tools.GuardResult, error) {
	guard := l.runtimeGuard
	if guard == nil {
		return tools.GuardResult{Action: tools.GuardAllow}, nil
	}

	now := l.nowUTC()
	mode, ttlExpired := guard.effectiveMode(now)
	evaluator := policy.NewEvaluator(policy.Config{
		Mode:            mode,
		RequireApproval: guard.requireApproval,
	})
	decision := evaluator.Evaluate(policy.Input{ToolName: name})

	switch decision.Action {
	case policy.ActionAllow:
		l.appendAuditEvent(ctx, "policy_allow", "", name, fmt.Sprintf("mode=%s", mode))
		return tools.GuardResult{Action: tools.GuardAllow}, nil
	case policy.ActionDeny:
		msg := strings.TrimSpace(decision.Reason)
		if msg == "" {
			msg = "blocked by policy"
		}
		l.appendAuditEvent(ctx, "policy_deny", "", name, msg)
		return tools.GuardResult{Action: tools.GuardDeny, Message: msg}, nil
	case policy.ActionRequireApproval:
		if _, err := guard.approvalService.ExpirePending(); err != nil {
			return tools.GuardResult{}, err
		}

		normalizedArgs := normalizeArgsJSON(argsJSON)
		approvedReq, pendingReq, err := guard.findMatchingRequests(name, normalizedArgs)
		if err != nil {
			return tools.GuardResult{}, err
		}
		if approvedReq != nil {
			l.appendAuditEvent(ctx, "approval_granted", approvedReq.ID, name, "matched approved request")
			return tools.GuardResult{Action: tools.GuardAllow}, nil
		}
		if pendingReq != nil {
			msg := fmt.Sprintf("approval required: id=%s (already pending)", pendingReq.ID)
			l.appendAuditEvent(ctx, "approval_pending", pendingReq.ID, name, "already pending")
			return tools.GuardResult{Action: tools.GuardRequireApproval, Message: msg}, nil
		}

		reason := fmt.Sprintf("policy mode %s requires approval", mode)
		if ttlExpired {
			reason = "policy off_ttl expired; strict mode restored"
		}

		req, err := guard.approvalService.Create(approval.CreateInput{
			ToolName: strings.TrimSpace(name),
			ArgsJSON: normalizedArgs,
			Reason:   reason,
		})
		if err != nil {
			return tools.GuardResult{}, err
		}

		msg := fmt.Sprintf("approval required: id=%s (run: golem approval approve %s --by <name>)", req.ID, req.ID)
		l.appendAuditEvent(ctx, "approval_pending", req.ID, name, reason)
		return tools.GuardResult{Action: tools.GuardRequireApproval, Message: msg}, nil
	default:
		msg := fmt.Sprintf("unknown policy decision: %s", decision.Action)
		l.appendAuditEvent(ctx, "policy_deny", "", name, msg)
		return tools.GuardResult{Action: tools.GuardDeny, Message: msg}, nil
	}
}

func (l *Loop) auditToolExecution(ctx context.Context, toolName, result string, err error) {
	if l.runtimeGuard == nil || l.runtimeGuard.auditWriter == nil {
		return
	}

	status := "success"
	normalizedResult := strings.ToLower(strings.TrimSpace(result))
	if strings.HasPrefix(normalizedResult, "pending approval") {
		status = "pending_approval"
	} else if err != nil || strings.HasPrefix(result, "Error:") {
		status = "error"
	}
	l.appendAuditEvent(ctx, "tool_execution", "", toolName, status)
}

func (l *Loop) appendAuditEvent(ctx context.Context, eventType, requestID, toolName, result string) {
	if l.runtimeGuard == nil || l.runtimeGuard.auditWriter == nil {
		return
	}

	reqID := strings.TrimSpace(requestID)
	if reqID == "" {
		reqID = tools.InvocationFromContext(ctx).RequestID
	}

	event := audit.Event{
		Time:      l.nowUTC(),
		Type:      strings.TrimSpace(eventType),
		RequestID: reqID,
		Tool:      strings.TrimSpace(toolName),
		Result:    strings.TrimSpace(result),
	}

	if err := l.runtimeGuard.auditWriter.Append(event); err != nil {
		slog.Warn("failed to append audit event", "type", event.Type, "tool", event.Tool, "error", err)
	}
}

func (l *Loop) nowUTC() time.Time {
	if l.now != nil {
		return l.now().UTC()
	}
	return time.Now().UTC()
}

func (g *runtimeGuard) effectiveMode(now time.Time) (policy.Mode, bool) {
	mode := g.baseMode
	if mode == policy.ModeOff && !g.offUntil.IsZero() && !now.Before(g.offUntil) {
		return policy.ModeStrict, true
	}
	return mode, false
}

func (g *runtimeGuard) findMatchingRequests(toolName, argsJSON string) (*approval.Request, *approval.Request, error) {
	requests, err := g.approvalService.List(approval.Query{ToolName: strings.TrimSpace(toolName)})
	if err != nil {
		return nil, nil, err
	}

	var approvedReq *approval.Request
	var pendingReq *approval.Request
	for i := range requests {
		req := requests[i]
		if !strings.EqualFold(strings.TrimSpace(req.ToolName), strings.TrimSpace(toolName)) {
			continue
		}
		if normalizeArgsJSON(req.ArgsJSON) != argsJSON {
			continue
		}

		switch req.Status {
		case approval.StatusApproved:
			if approvedReq == nil || req.DecidedAt.After(approvedReq.DecidedAt) {
				copied := req
				approvedReq = &copied
			}
		case approval.StatusPending:
			if pendingReq == nil || req.RequestedAt.After(pendingReq.RequestedAt) {
				copied := req
				pendingReq = &copied
			}
		}
	}

	return approvedReq, pendingReq, nil
}

func normalizeArgsJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}

	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(trimmed)); err != nil {
		return trimmed
	}
	return buf.String()
}
