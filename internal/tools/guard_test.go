package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type countingTool struct {
	runs int
}

func (m *countingTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "guarded_tool",
		Desc: "Tool used in guard tests",
	}, nil
}

func (m *countingTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	m.runs++
	return "tool ran", nil
}

func TestRegistry_Execute_DeniedByGuard(t *testing.T) {
	reg := NewRegistry()
	mock := &countingTool{}
	if err := reg.Register(mock); err != nil {
		t.Fatalf("Register error: %v", err)
	}

	reg.SetGuard(func(ctx context.Context, name, argsJSON string) (GuardResult, error) {
		return GuardResult{Action: GuardDeny, Message: "blocked by policy"}, nil
	})

	result, err := reg.Execute(context.Background(), "guarded_tool", `{}`)
	if err == nil {
		t.Fatal("expected deny error")
	}
	if result != "" {
		t.Fatalf("expected empty result, got %q", result)
	}
	if !strings.Contains(err.Error(), "blocked by policy") {
		t.Fatalf("expected deny message in error, got: %v", err)
	}
	if mock.runs != 0 {
		t.Fatalf("expected tool not to run, ran %d times", mock.runs)
	}
}

func TestRegistry_Execute_ApprovalPending(t *testing.T) {
	reg := NewRegistry()
	mock := &countingTool{}
	if err := reg.Register(mock); err != nil {
		t.Fatalf("Register error: %v", err)
	}

	reg.SetGuard(func(ctx context.Context, name, argsJSON string) (GuardResult, error) {
		return GuardResult{Action: GuardRequireApproval, Message: "approval #123 required"}, nil
	})

	result, err := reg.Execute(context.Background(), "guarded_tool", `{}`)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "pending approval") {
		t.Fatalf("expected pending approval result, got: %q", result)
	}
	if !strings.Contains(result, "approval #123 required") {
		t.Fatalf("expected guard message in result, got: %q", result)
	}
	if mock.runs != 0 {
		t.Fatalf("expected tool not to run, ran %d times", mock.runs)
	}
}

func TestRegistry_Execute_Allowed(t *testing.T) {
	reg := NewRegistry()
	mock := &countingTool{}
	if err := reg.Register(mock); err != nil {
		t.Fatalf("Register error: %v", err)
	}

	reg.SetGuard(func(ctx context.Context, name, argsJSON string) (GuardResult, error) {
		return GuardResult{Action: GuardAllow}, nil
	})

	result, err := reg.Execute(context.Background(), "guarded_tool", `{}`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result != "tool ran" {
		t.Fatalf("expected tool result, got: %q", result)
	}
	if mock.runs != 1 {
		t.Fatalf("expected tool to run once, ran %d times", mock.runs)
	}
}

func TestRegistry_Execute_GuardErrorPropagates(t *testing.T) {
	reg := NewRegistry()
	mock := &countingTool{}
	if err := reg.Register(mock); err != nil {
		t.Fatalf("Register error: %v", err)
	}

	reg.SetGuard(func(ctx context.Context, name, argsJSON string) (GuardResult, error) {
		return GuardResult{}, errors.New("guard backend unavailable")
	})

	result, err := reg.Execute(context.Background(), "guarded_tool", `{}`)
	if err == nil {
		t.Fatal("expected guard error")
	}
	if result != "" {
		t.Fatalf("expected empty result on guard error, got %q", result)
	}
	if !strings.Contains(err.Error(), "guard backend unavailable") {
		t.Fatalf("unexpected guard error: %v", err)
	}
	if mock.runs != 0 {
		t.Fatalf("expected tool not to run on guard error, ran %d times", mock.runs)
	}
}
