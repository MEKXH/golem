package tools

import (
    "context"
    "testing"

    "github.com/cloudwego/eino/schema"
)

// Mock tool for testing
type mockTool struct{}

func (m *mockTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "mock_tool",
        Desc: "A mock tool for testing",
    }, nil
}

func (m *mockTool) InvokableRun(ctx context.Context, args string, opts ...any) (string, error) {
    return "mock result", nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
    reg := NewRegistry()

    err := reg.Register(&mockTool{})
    if err != nil {
        t.Fatalf("Register error: %v", err)
    }

    tool, ok := reg.Get("mock_tool")
    if !ok {
        t.Fatal("expected to find mock_tool")
    }
    if tool == nil {
        t.Fatal("tool is nil")
    }
}
