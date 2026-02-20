package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type GuardAction string

const (
	GuardAllow           GuardAction = "allow"
	GuardDeny            GuardAction = "deny"
	GuardRequireApproval GuardAction = "require_approval"
)

type GuardResult struct {
	Action  GuardAction
	Message string
}

type GuardFunc func(ctx context.Context, name, argsJSON string) (GuardResult, error)

// Registry manages tools by name
type Registry struct {
	mu    sync.RWMutex
	tools map[string]tool.InvokableTool
	guard GuardFunc
}

// NewRegistry creates a new registry
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]tool.InvokableTool)}
}

// Register adds a tool to registry
func (r *Registry) Register(t tool.InvokableTool) error {
	info, err := t.Info(context.Background())
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
	r.tools[info.Name] = t
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (tool.InvokableTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// SetGuard sets a pre-execution guard for tool execution.
func (r *Registry) SetGuard(fn GuardFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.guard = fn
}

func (r *Registry) getGuard() GuardFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.guard
}

// GetToolInfos returns all tool schemas for ChatModel binding
func (r *Registry) GetToolInfos(ctx context.Context) ([]*schema.ToolInfo, error) {
	r.mu.RLock()
	toolsList := make([]tool.InvokableTool, 0, len(r.tools))
	for _, t := range r.tools {
		toolsList = append(toolsList, t)
	}
	r.mu.RUnlock()

	infos := make([]*schema.ToolInfo, 0, len(toolsList))
	for _, t := range toolsList {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// Execute runs a tool by name
func (r *Registry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	if guard := r.getGuard(); guard != nil {
		result, err := guard(ctx, name, argsJSON)
		if err != nil {
			return "", err
		}

		switch result.Action {
		case "", GuardAllow:
			// Continue to tool execution.
		case GuardDeny:
			msg := strings.TrimSpace(result.Message)
			if msg == "" {
				msg = "tool execution denied"
			}
			return "", fmt.Errorf("tool execution denied: %s", msg)
		case GuardRequireApproval:
			msg := strings.TrimSpace(result.Message)
			if msg == "" {
				return "pending approval", nil
			}
			return "pending approval: " + msg, nil
		default:
			return "", fmt.Errorf("unknown guard action: %s", result.Action)
		}
	}

	return t.InvokableRun(ctx, argsJSON)
}

// Names returns all registered tool names
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// List returns all tools
func (r *Registry) List() []tool.InvokableTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tool.InvokableTool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}
