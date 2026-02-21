package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/session"
	"github.com/MEKXH/golem/internal/tools"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// slowTool is a mock tool that sleeps.
type slowTool struct {
	delay time.Duration
}

func (t *slowTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "slow_tool",
		Desc: "A slow tool",
	}, nil
}

func (t *slowTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	time.Sleep(t.delay)
	return "done", nil
}

// parallelMockModel returns multiple tool calls.
type parallelMockModel struct {
	callCount int
	toolCount int
}

func (m *parallelMockModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.callCount++
	if m.callCount == 1 {
		toolCalls := make([]schema.ToolCall, m.toolCount)
		for i := 0; i < m.toolCount; i++ {
			toolCalls[i] = schema.ToolCall{
				ID: fmt.Sprintf("call_%d", i),
				Function: schema.FunctionCall{
					Name:      "slow_tool",
					Arguments: "{}",
				},
			}
		}
		return &schema.Message{
			Role:      schema.Assistant,
			Content:   "",
			ToolCalls: toolCalls,
		}, nil
	}
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "Final response",
	}, nil
}

func (m *parallelMockModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *parallelMockModel) BindTools(toolInfos []*schema.ToolInfo) error {
	return nil
}

func TestLoop_ParallelToolExecution(t *testing.T) {
	delay := 100 * time.Millisecond
	toolCount := 3

	mockModel := &parallelMockModel{toolCount: toolCount}

	tmpDir := t.TempDir()
	loop := &Loop{
		bus:           bus.NewMessageBus(1),
		model:         mockModel,
		tools:         tools.NewRegistry(),
		sessions:      session.NewManager(tmpDir),
		context:       NewContextBuilder(tmpDir),
		maxIterations: 10,
		workspacePath: tmpDir,
	}

	// Register slow tool
	if err := loop.tools.Register(&slowTool{delay: delay}); err != nil {
		t.Fatalf("failed to register slow tool: %v", err)
	}

	start := time.Now()
	_, err := loop.ProcessDirect(context.Background(), "trigger tools")
	if err != nil {
		t.Fatalf("ProcessDirect error: %v", err)
	}
	duration := time.Since(start)

	// Currently, it's sequential. 3 * 100ms = 300ms.
	// Parallel would be ~100ms.

	expectedParallelTime := delay * 2 // 200ms

	if duration < expectedParallelTime {
		t.Logf("PASS: Execution took %v (parallel)", duration)
	} else {
		// This is expected failure for now
		t.Logf("FAIL: Execution took %v (sequential)", duration)
		// We don't fail the test here so I can proceed with the plan,
		// but typically a reproduction test should fail first.
		// However, since I'm implementing a performance improvement, the test
		// is meant to demonstrate the difference.
		// I'll make it fail if it's too slow so I can see it fail.
		t.Errorf("Expected execution time < %v, got %v", expectedParallelTime, duration)
	}
}
