package approval

import (
	"testing"
	"time"
)

func TestService_CreateAndApproveFlow(t *testing.T) {
	workspace := t.TempDir()
	svc := NewService(workspace)
	fixedNow := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	created, err := svc.Create(CreateInput{
		ToolName: "exec",
		ArgsJSON: `{"command":"echo hi"}`,
		Reason:   "needs shell access",
		TTL:      5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.Status != StatusPending {
		t.Fatalf("expected status %q, got %q", StatusPending, created.Status)
	}
	if created.RequestedAt != fixedNow {
		t.Fatalf("unexpected requested_at: %s", created.RequestedAt)
	}
	if created.ExpiresAt.IsZero() {
		t.Fatal("expected non-zero expires_at")
	}

	svc.now = func() time.Time { return fixedNow.Add(2 * time.Minute) }
	approved, err := svc.Approve(created.ID, DecisionInput{
		DecidedBy: "owner",
		Note:      "safe command",
	})
	if err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Fatalf("expected status %q, got %q", StatusApproved, approved.Status)
	}
	if approved.DecidedBy != "owner" {
		t.Fatalf("unexpected decided_by: %q", approved.DecidedBy)
	}
	if approved.DecisionNote != "safe command" {
		t.Fatalf("unexpected decision_note: %q", approved.DecisionNote)
	}
	if approved.DecidedAt.IsZero() {
		t.Fatal("expected non-zero decided_at")
	}

	approvedList, err := svc.List(Query{Status: StatusApproved})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(approvedList) != 1 {
		t.Fatalf("expected 1 approved request, got %d", len(approvedList))
	}

	svcReloaded := NewService(workspace)
	persistedApproved, err := svcReloaded.List(Query{Status: StatusApproved})
	if err != nil {
		t.Fatalf("List after reload error: %v", err)
	}
	if len(persistedApproved) != 1 {
		t.Fatalf("expected 1 approved request after reload, got %d", len(persistedApproved))
	}
}

func TestService_RejectFlow(t *testing.T) {
	svc := NewService(t.TempDir())
	fixedNow := time.Date(2026, 2, 15, 11, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	created, err := svc.Create(CreateInput{
		ToolName: "write_file",
		ArgsJSON: `{"path":"README.md"}`,
		Reason:   "change docs",
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	svc.now = func() time.Time { return fixedNow.Add(time.Minute) }
	rejected, err := svc.Reject(created.ID, DecisionInput{
		DecidedBy: "owner",
		Note:      "not needed",
	})
	if err != nil {
		t.Fatalf("Reject error: %v", err)
	}
	if rejected.Status != StatusRejected {
		t.Fatalf("expected status %q, got %q", StatusRejected, rejected.Status)
	}
	if rejected.DecisionNote != "not needed" {
		t.Fatalf("unexpected decision_note: %q", rejected.DecisionNote)
	}

	rejectedList, err := svc.List(Query{Status: StatusRejected})
	if err != nil {
		t.Fatalf("List rejected error: %v", err)
	}
	if len(rejectedList) != 1 {
		t.Fatalf("expected 1 rejected request, got %d", len(rejectedList))
	}
}

func TestService_ExpirePendingByTTL(t *testing.T) {
	svc := NewService(t.TempDir())
	baseNow := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return baseNow }

	expiringSoon, err := svc.Create(CreateInput{
		ToolName: "exec",
		ArgsJSON: `{"command":"ls"}`,
		Reason:   "inspection",
		TTL:      30 * time.Second,
	})
	if err != nil {
		t.Fatalf("Create expiringSoon error: %v", err)
	}

	stillPending, err := svc.Create(CreateInput{
		ToolName: "exec",
		ArgsJSON: `{"command":"pwd"}`,
		Reason:   "inspection",
		TTL:      5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Create stillPending error: %v", err)
	}

	svc.now = func() time.Time { return baseNow.Add(31 * time.Second) }
	expired, err := svc.ExpirePending()
	if err != nil {
		t.Fatalf("ExpirePending error: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired request, got %d", len(expired))
	}
	if expired[0].ID != expiringSoon.ID {
		t.Fatalf("expected expired id %q, got %q", expiringSoon.ID, expired[0].ID)
	}
	if expired[0].Status != StatusExpired {
		t.Fatalf("expected status %q, got %q", StatusExpired, expired[0].Status)
	}
	if expired[0].DecidedBy != "system" {
		t.Fatalf("expected decided_by system, got %q", expired[0].DecidedBy)
	}
	if expired[0].DecisionNote == "" {
		t.Fatal("expected non-empty decision note for expired request")
	}

	pending, err := svc.List(Query{Status: StatusPending})
	if err != nil {
		t.Fatalf("List pending error: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending request, got %d", len(pending))
	}
	if pending[0].ID != stillPending.ID {
		t.Fatalf("expected pending id %q, got %q", stillPending.ID, pending[0].ID)
	}
}

func TestService_CreateRejectsEmptyToolName(t *testing.T) {
	svc := NewService(t.TempDir())
	if _, err := svc.Create(CreateInput{ToolName: "   "}); err == nil {
		t.Fatal("expected create to fail for empty tool name")
	}
}

func TestService_ApproveAlreadyDecidedFails(t *testing.T) {
	svc := NewService(t.TempDir())
	now := time.Date(2026, 2, 15, 13, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	req, err := svc.Create(CreateInput{ToolName: "exec", ArgsJSON: `{}`, TTL: time.Minute})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if _, err := svc.Approve(req.ID, DecisionInput{DecidedBy: "owner"}); err != nil {
		t.Fatalf("first approve error: %v", err)
	}
	if _, err := svc.Approve(req.ID, DecisionInput{DecidedBy: "owner"}); err == nil {
		t.Fatal("expected second approve to fail for non-pending request")
	}
}

func TestService_CreateDefaultTTLApplied(t *testing.T) {
	svc := NewService(t.TempDir())
	now := time.Date(2026, 2, 15, 14, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	req, err := svc.Create(CreateInput{ToolName: "exec", ArgsJSON: `{}`, TTL: 0})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if !req.ExpiresAt.Equal(now.Add(defaultTTL)) {
		t.Fatalf("expected expires_at %s, got %s", now.Add(defaultTTL), req.ExpiresAt)
	}
}
