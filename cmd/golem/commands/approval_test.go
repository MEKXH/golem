package commands

import (
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/approval"
	"github.com/MEKXH/golem/internal/config"
)

func TestApprovalList_ShowsPendingOnly(t *testing.T) {
	workspacePath := prepareApprovalWorkspace(t)

	svc := approval.NewService(workspacePath)
	pending, err := svc.Create(approval.CreateInput{
		ToolName: "exec",
		ArgsJSON: `{"command":"echo hi"}`,
	})
	if err != nil {
		t.Fatalf("Create pending approval: %v", err)
	}
	approved, err := svc.Create(approval.CreateInput{
		ToolName: "write_file",
		ArgsJSON: `{"path":"README.md"}`,
	})
	if err != nil {
		t.Fatalf("Create approval to approve: %v", err)
	}
	if _, err := svc.Approve(approved.ID, approval.DecisionInput{
		DecidedBy: "owner",
		Note:      "safe",
	}); err != nil {
		t.Fatalf("Approve approval: %v", err)
	}

	output := captureOutput(t, func() {
		if err := runApprovalList(nil, nil); err != nil {
			t.Fatalf("runApprovalList: %v", err)
		}
	})

	if !strings.Contains(output, pending.ID) {
		t.Fatalf("expected pending id %q in output, got: %s", pending.ID, output)
	}
	if strings.Contains(output, approved.ID) {
		t.Fatalf("did not expect approved id %q in output, got: %s", approved.ID, output)
	}
}

func TestApprovalList_NoPending(t *testing.T) {
	_ = prepareApprovalWorkspace(t)
	output := captureOutput(t, func() {
		if err := runApprovalList(nil, nil); err != nil {
			t.Fatalf("runApprovalList: %v", err)
		}
	})
	if !strings.Contains(output, "No pending approvals.") {
		t.Fatalf("expected no-pending message, got: %s", output)
	}
}

func TestApprovalApprove_UpdatesDecision(t *testing.T) {
	workspacePath := prepareApprovalWorkspace(t)

	svc := approval.NewService(workspacePath)
	req, err := svc.Create(approval.CreateInput{
		ToolName: "exec",
		ArgsJSON: `{"command":"pwd"}`,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	cmd := newApprovalApproveCmd()
	if err := cmd.Flags().Set("by", "owner"); err != nil {
		t.Fatalf("set --by: %v", err)
	}
	if err := cmd.Flags().Set("note", "looks good"); err != nil {
		t.Fatalf("set --note: %v", err)
	}

	output := captureOutput(t, func() {
		if err := runApprovalApprove(cmd, []string{req.ID}); err != nil {
			t.Fatalf("runApprovalApprove: %v", err)
		}
	})

	if !strings.Contains(output, "approved") {
		t.Fatalf("expected approved output, got: %s", output)
	}

	approved, err := svc.List(approval.Query{ID: req.ID, Status: approval.StatusApproved})
	if err != nil {
		t.Fatalf("List approved: %v", err)
	}
	if len(approved) != 1 {
		t.Fatalf("expected 1 approved request, got %d", len(approved))
	}
	if approved[0].DecidedBy != "owner" {
		t.Fatalf("expected decided_by owner, got %q", approved[0].DecidedBy)
	}
	if approved[0].DecisionNote != "looks good" {
		t.Fatalf("expected decision note, got %q", approved[0].DecisionNote)
	}
}

func TestApprovalApprove_RequiresBy(t *testing.T) {
	workspacePath := prepareApprovalWorkspace(t)
	svc := approval.NewService(workspacePath)
	req, err := svc.Create(approval.CreateInput{
		ToolName: "exec",
		ArgsJSON: `{"command":"pwd"}`,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	cmd := newApprovalApproveCmd()
	if err := runApprovalApprove(cmd, []string{req.ID}); err == nil {
		t.Fatal("expected error when --by is missing")
	}
}

func TestApprovalReject_UpdatesDecision(t *testing.T) {
	workspacePath := prepareApprovalWorkspace(t)

	svc := approval.NewService(workspacePath)
	req, err := svc.Create(approval.CreateInput{
		ToolName: "exec",
		ArgsJSON: `{"command":"rm -rf /tmp/demo"}`,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	cmd := newApprovalRejectCmd()
	if err := cmd.Flags().Set("by", "reviewer"); err != nil {
		t.Fatalf("set --by: %v", err)
	}
	if err := cmd.Flags().Set("note", "unsafe"); err != nil {
		t.Fatalf("set --note: %v", err)
	}

	output := captureOutput(t, func() {
		if err := runApprovalReject(cmd, []string{req.ID}); err != nil {
			t.Fatalf("runApprovalReject: %v", err)
		}
	})

	if !strings.Contains(output, "rejected") {
		t.Fatalf("expected rejected output, got: %s", output)
	}

	rejected, err := svc.List(approval.Query{ID: req.ID, Status: approval.StatusRejected})
	if err != nil {
		t.Fatalf("List rejected: %v", err)
	}
	if len(rejected) != 1 {
		t.Fatalf("expected 1 rejected request, got %d", len(rejected))
	}
	if rejected[0].DecidedBy != "reviewer" {
		t.Fatalf("expected decided_by reviewer, got %q", rejected[0].DecidedBy)
	}
	if rejected[0].DecisionNote != "unsafe" {
		t.Fatalf("expected decision note unsafe, got %q", rejected[0].DecisionNote)
	}
}

func TestApprovalCommand_RegisteredInRoot(t *testing.T) {
	root := NewRootCmd()
	found, _, err := root.Find([]string{"approval", "list"})
	if err != nil {
		t.Fatalf("find approval list command: %v", err)
	}
	if found == nil || found.Name() != "list" {
		t.Fatalf("expected list command, got %#v", found)
	}
}

func prepareApprovalWorkspace(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		t.Fatalf("WorkspacePathChecked: %v", err)
	}
	return workspacePath
}
