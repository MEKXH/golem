package agent

import (
    "context"
    "strings"
    "testing"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/config"
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
