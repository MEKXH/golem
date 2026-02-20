package tools

import (
	"context"
	"strings"
	"testing"
)

type fakeWorkflowExecutor struct {
	lastReq WorkflowRequest
	out     string
	err     error
}

func (f *fakeWorkflowExecutor) RunWorkflow(ctx context.Context, req WorkflowRequest) (string, error) {
	f.lastReq = req
	if f.err != nil {
		return "", f.err
	}
	if strings.TrimSpace(f.out) == "" {
		return "workflow done", nil
	}
	return f.out, nil
}

func TestWorkflowTool_BuildsRequestFromInvocationContext(t *testing.T) {
	exec := &fakeWorkflowExecutor{out: "workflow summary"}
	tool, err := NewWorkflowTool(exec)
	if err != nil {
		t.Fatalf("NewWorkflowTool: %v", err)
	}

	ctx := WithInvocationContext(context.Background(), InvocationContext{
		Channel:   "discord",
		ChatID:    "room-7",
		SenderID:  "alice",
		RequestID: "req-77",
	})
	_, err = tool.InvokableRun(ctx, `{"goal":"triage incident","mode":"parallel","subtasks":["check logs","verify queue"]}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}

	if exec.lastReq.OriginChannel != "discord" || exec.lastReq.OriginChatID != "room-7" {
		t.Fatalf("unexpected route binding: %+v", exec.lastReq)
	}
	if exec.lastReq.OriginSenderID != "alice" || exec.lastReq.RequestID != "req-77" {
		t.Fatalf("unexpected sender/request binding: %+v", exec.lastReq)
	}
	if exec.lastReq.Goal != "triage incident" || exec.lastReq.Mode != "parallel" {
		t.Fatalf("unexpected workflow payload: %+v", exec.lastReq)
	}
	if len(exec.lastReq.Subtasks) != 2 {
		t.Fatalf("expected two subtasks, got %+v", exec.lastReq.Subtasks)
	}
}

func TestWorkflowTool_RequiresGoal(t *testing.T) {
	exec := &fakeWorkflowExecutor{}
	tool, err := NewWorkflowTool(exec)
	if err != nil {
		t.Fatalf("NewWorkflowTool: %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"mode":"parallel","subtasks":["x"]}`)
	if err == nil {
		t.Fatal("expected goal validation error")
	}
}
