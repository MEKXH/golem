// Package tools 实现 Golem 的内置工具体系，并提供工具的注册、守卫与执行管理。
package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GuardAction 定义守卫函数对工具执行的决策动作。
type GuardAction string

const (
	GuardAllow           GuardAction = "allow"            // 允许执行
	GuardDeny            GuardAction = "deny"             // 拒绝执行
	GuardRequireApproval GuardAction = "require_approval" // 需要人工审批
)

// GuardResult 包含守卫决策的结果及相关提示消息。
type GuardResult struct {
	Action  GuardAction // 决策动作
	Message string      // 决策原因或给用户的提示
}

// GuardFunc 定义了工具执行前的守卫函数原型。
type GuardFunc func(ctx context.Context, name, argsJSON string) (GuardResult, error)

// Registry 按名称统一管理所有可调用的工具实例。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]tool.InvokableTool // 工具名称到实例的映射
	guard GuardFunc                     // 执行前置守卫逻辑
}

// NewRegistry 创建并初始化一个新的工具注册表。
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]tool.InvokableTool)}
}

// Register 向注册表中添加一个新的工具实例。如果同名工具已存在，将返回错误。
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

// Get 根据名称从注册表中检索工具实例。
func (r *Registry) Get(name string) (tool.InvokableTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// SetGuard 设置全局工具执行守卫函数，用于权限控制或审计。
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

// GetToolInfos 返回所有已注册工具的元数据（Schema），通常用于 LLM 的工具绑定。
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

// Execute 根据名称运行指定的工具。在执行前会自动触发守卫函数进行检查。
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
			// 守卫允许，继续执行
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

// Names 返回所有已注册工具的名称列表。
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// List 返回所有已注册工具的实例列表。
func (r *Registry) List() []tool.InvokableTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tool.InvokableTool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}
