package tools

import (
	"context"
	"strings"
	"testing"
)

type fakeSubagentExecutor struct {
	lastSpawnReq SubagentRequest
	lastSyncReq  SubagentRequest
	spawnID      string
	syncResult   string
}

func (f *fakeSubagentExecutor) Spawn(ctx context.Context, req SubagentRequest) (string, error) {
	f.lastSpawnReq = req
	if f.spawnID == "" {
		f.spawnID = "subagent-1"
	}
	return f.spawnID, nil
}

func (f *fakeSubagentExecutor) RunSync(ctx context.Context, req SubagentRequest) (string, error) {
	f.lastSyncReq = req
	if f.syncResult == "" {
		f.syncResult = "done"
	}
	return f.syncResult, nil
}

func TestSpawnTool_UsesInvocationDefaults(t *testing.T) {
	exec := &fakeSubagentExecutor{spawnID: "subagent-9"}
	tool, err := NewSpawnTool(exec)
	if err != nil {
		t.Fatalf("NewSpawnTool: %v", err)
	}

	ctx := WithInvocationContext(context.Background(), InvocationContext{
		Channel:   "discord",
		ChatID:    "room-1",
		SenderID:  "alice",
		RequestID: "req-42",
	})

	out, err := tool.InvokableRun(ctx, `{"task":"collect logs","label":"ops"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if !strings.Contains(out, "subagent-9") {
		t.Fatalf("expected task id in result, got: %s", out)
	}
	if exec.lastSpawnReq.OriginChannel != "discord" || exec.lastSpawnReq.OriginChatID != "room-1" {
		t.Fatalf("unexpected origin route: %+v", exec.lastSpawnReq)
	}
	if exec.lastSpawnReq.OriginSenderID != "alice" || exec.lastSpawnReq.RequestID != "req-42" {
		t.Fatalf("unexpected sender/request binding: %+v", exec.lastSpawnReq)
	}
}

func TestSubagentTool_RunSyncReturnsResult(t *testing.T) {
	exec := &fakeSubagentExecutor{syncResult: "analysis complete"}
	tool, err := NewSubagentTool(exec)
	if err != nil {
		t.Fatalf("NewSubagentTool: %v", err)
	}

	out, err := tool.InvokableRun(context.Background(), `{"task":"analyze test failures","label":"qa"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if !strings.Contains(out, "analysis complete") {
		t.Fatalf("expected sync result in output, got: %s", out)
	}
	if exec.lastSyncReq.Task != "analyze test failures" || exec.lastSyncReq.Label != "qa" {
		t.Fatalf("unexpected sync request payload: %+v", exec.lastSyncReq)
	}
}
