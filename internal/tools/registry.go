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

// Registry 按名称管理工具
type Registry struct {
	mu    sync.RWMutex
	tools map[string]tool.InvokableTool
	guard GuardFunc
}

// NewRegistry 创建一个新的注册表
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]tool.InvokableTool)}
}

// Register 向注册表添加工具
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

// Get 按名称获取工具
func (r *Registry) Get(name string) (tool.InvokableTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// SetGuard 设置工具执行前的守卫函数。
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

// GetToolInfos 返回所有工具的 Schema，用于 ChatModel 绑定
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

// Execute 按名称运行工具
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

// Names 返回所有已注册的工具名称
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// List 返回所有工具
func (r *Registry) List() []tool.InvokableTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tool.InvokableTool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}
