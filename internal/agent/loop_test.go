package agent

import (
    "context"
    "fmt"
    "strings"
    "testing"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/schema"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/config"
    "github.com/MEKXH/golem/internal/session"
    "github.com/MEKXH/golem/internal/tools"
)

type mockChatModel struct {
    bindCalls  int
    boundTools int
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
    return &schema.Message{Role: schema.Assistant, Content: "ok"}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
    return nil, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
    m.bindCalls++
    m.boundTools = len(tools)
    return nil
}

func TestNewLoop(t *testing.T) {
    cfg := config.DefaultConfig()
    msgBus := bus.NewMessageBus(10)

    loop, err := NewLoop(cfg, msgBus, nil)
    if err != nil {
        t.Fatalf("NewLoop error: %v", err)
    }
    if loop == nil {
        t.Fatal("expected non-nil Loop")
    }
    if loop.maxIterations != 20 {
        t.Errorf("expected maxIterations=20, got %d", loop.maxIterations)
    }
}

func TestNewLoop_InvalidWorkspaceMode(t *testing.T) {
    cfg := config.DefaultConfig()
    cfg.Agents.Defaults.WorkspaceMode = "path"
    cfg.Agents.Defaults.Workspace = ""
    msgBus := bus.NewMessageBus(10)

    if _, err := NewLoop(cfg, msgBus, nil); err == nil {
        t.Fatal("expected error")
    }
}

func TestContextBuilder_BuildSystemPrompt(t *testing.T) {
    tmpDir := t.TempDir()
    cb := NewContextBuilder(tmpDir)

    prompt := cb.BuildSystemPrompt()
    if !strings.Contains(prompt, "Golem") {
        t.Error("expected system prompt to contain 'Golem'")
    }
}

func TestProcessDirect_BindsTools(t *testing.T) {
    tmpDir := t.TempDir()
    t.Setenv("HOME", tmpDir)
    t.Setenv("USERPROFILE", tmpDir)

    cfg := config.DefaultConfig()
    msgBus := bus.NewMessageBus(1)
    model := &mockChatModel{}

    loop, err := NewLoop(cfg, msgBus, model)
    if err != nil {
        t.Fatalf("NewLoop error: %v", err)
    }
    if err := loop.RegisterDefaultTools(cfg); err != nil {
        t.Fatalf("RegisterDefaultTools error: %v", err)
    }

    if got := len(loop.tools.Names()); got == 0 {
        t.Fatalf("expected tools registered, got %d", got)
    }

    _, err = loop.ProcessDirect(context.Background(), "hi")
    if err != nil {
        t.Fatalf("ProcessDirect error: %v", err)
    }

    if model.bindCalls == 0 {
        t.Fatalf("expected BindTools to be called")
    }
    if model.boundTools == 0 {
        t.Fatalf("expected tools to be bound")
    }
}

// multiTurnMockModel returns a tool call on the first Generate call, then a final text response.
type multiTurnMockModel struct {
    callCount int
}

func (m *multiTurnMockModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
    m.callCount++
    if m.callCount == 1 {
        return &schema.Message{
            Role:    schema.Assistant,
            Content: "",
            ToolCalls: []schema.ToolCall{
                {
                    ID: "call_1",
                    Function: schema.FunctionCall{
                        Name:      "mock_tool",
                        Arguments: `{"input":"test"}`,
                    },
                },
            },
        }, nil
    }
    return &schema.Message{
        Role:    schema.Assistant,
        Content: "Final response",
    }, nil
}

func (m *multiTurnMockModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
    return nil, nil
}

func (m *multiTurnMockModel) BindTools(toolInfos []*schema.ToolInfo) error {
    return nil
}

// alwaysToolCallModel always returns a tool call, never a final response.
type alwaysToolCallModel struct {
    callCount int
}

func (m *alwaysToolCallModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
    m.callCount++
    return &schema.Message{
        Role:    schema.Assistant,
        Content: "",
        ToolCalls: []schema.ToolCall{
            {
                ID: "call_" + fmt.Sprintf("%d", m.callCount),
                Function: schema.FunctionCall{
                    Name:      "mock_tool",
                    Arguments: `{"input":"loop"}`,
                },
            },
        },
    }, nil
}

func (m *alwaysToolCallModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
    return nil, nil
}

func (m *alwaysToolCallModel) BindTools(toolInfos []*schema.ToolInfo) error {
    return nil
}

// testTool is a simple mock tool implementing tool.InvokableTool.
type testTool struct{}

func (t *testTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "mock_tool",
        Desc: "A test tool",
    }, nil
}

func (t *testTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
    return "tool executed successfully", nil
}

// newTestLoop creates a Loop with the given model, maxIterations, and a temp workspace.
func newTestLoop(t *testing.T, chatModel model.ChatModel, maxIterations int) *Loop {
    t.Helper()
    tmpDir := t.TempDir()
    return &Loop{
        bus:           bus.NewMessageBus(1),
        model:         chatModel,
        tools:         tools.NewRegistry(),
        sessions:      session.NewManager(tmpDir),
        context:       NewContextBuilder(tmpDir),
        maxIterations: maxIterations,
        workspacePath: tmpDir,
    }
}

func TestProcessDirect_WithToolCalls(t *testing.T) {
    mockModel := &multiTurnMockModel{}
    loop := newTestLoop(t, mockModel, 10)

    // Register the mock tool so the loop can execute it.
    if err := loop.tools.Register(&testTool{}); err != nil {
        t.Fatalf("failed to register mock tool: %v", err)
    }

    result, err := loop.ProcessDirect(context.Background(), "test message")
    if err != nil {
        t.Fatalf("ProcessDirect returned error: %v", err)
    }

    if result != "Final response" {
        t.Errorf("expected result %q, got %q", "Final response", result)
    }

    if mockModel.callCount != 2 {
        t.Errorf("expected model to be called 2 times, got %d", mockModel.callCount)
    }
}

func TestProcessDirect_MaxIterations(t *testing.T) {
    mockModel := &alwaysToolCallModel{}
    loop := newTestLoop(t, mockModel, 2)

    // Register the mock tool so tool execution does not fail.
    if err := loop.tools.Register(&testTool{}); err != nil {
        t.Fatalf("failed to register mock tool: %v", err)
    }

    result, err := loop.ProcessDirect(context.Background(), "test message")
    if err != nil {
        t.Fatalf("ProcessDirect returned error: %v", err)
    }

    // When maxIterations is exhausted and the model never returns a final text response,
    // the loop falls through with an empty finalContent, which gets replaced by "Processing complete."
    if result != "Processing complete." {
        t.Errorf("expected result %q, got %q", "Processing complete.", result)
    }

    if mockModel.callCount != 2 {
        t.Errorf("expected model to be called exactly 2 times (maxIterations), got %d", mockModel.callCount)
    }
}
