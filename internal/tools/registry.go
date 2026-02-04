package tools

import (
    "context"
    "fmt"
    "sync"

    "github.com/cloudwego/eino/schema"
)

// Tool represents an executable tool
// Eino tools implement ToolInfo + InvokableRun

type Tool interface {
    Info(ctx context.Context) (*schema.ToolInfo, error)
    InvokableRun(ctx context.Context, args string, opts ...any) (string, error)
}

// Registry manages tools by name
type Registry struct {
    mu    sync.RWMutex
    tools map[string]Tool
}

// NewRegistry creates a new registry
func NewRegistry() *Registry {
    return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to registry
func (r *Registry) Register(tool Tool) error {
    info, err := tool.Info(context.Background())
    if err != nil {
        return err
    }
    if info == nil || info.Name == "" {
        return fmt.Errorf("tool info missing name")
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.tools[info.Name]; exists {
        return fmt.Errorf("tool already registered: %s", info.Name)
    }
    r.tools[info.Name] = tool
    return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tool, ok := r.tools[name]
    return tool, ok
}

// List returns all tools
func (r *Registry) List() []Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    result := make([]Tool, 0, len(r.tools))
    for _, tool := range r.tools {
        result = append(result, tool)
    }
    return result
}
