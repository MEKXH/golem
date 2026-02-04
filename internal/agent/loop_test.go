package agent

import (
    "strings"
    "testing"

    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/config"
)

func TestNewLoop(t *testing.T) {
    cfg := config.DefaultConfig()
    msgBus := bus.NewMessageBus(10)

    loop := NewLoop(cfg, msgBus, nil)
    if loop == nil {
        t.Fatal("expected non-nil Loop")
    }
    if loop.maxIterations != 20 {
        t.Errorf("expected maxIterations=20, got %d", loop.maxIterations)
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
