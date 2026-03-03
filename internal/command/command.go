// Package command 定义了 Golem 斜杠命令（Slash Commands）的接口、注册与分发逻辑。
package command

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/metrics"
	"github.com/MEKXH/golem/internal/session"
)

// Env 携带斜杠命令执行时的上下文环境信息。
type Env struct {
	Channel       string                  // 消息来源通道
	ChatID        string                  // 聊天 ID
	SenderID      string                  // 发送者 ID
	SessionKey    string                  // 唯一的会话标识符
	Sessions      *session.Manager        // 会话管理器，用于操作聊天历史
	WorkspacePath string                  // 工作区根路径
	Config        *config.Config          // 全局配置实例
	Metrics       *metrics.RuntimeMetrics // 运行时指标记录器
	ListCommands  func() []Command        // 用于 /help 获取所有可用命令的回调函数
}

// Result 封装了斜杠命令执行后的输出内容。
type Result struct {
	Content string // 返回给用户的文本消息
}

// Command 是所有斜杠命令必须实现的接口。
type Command interface {
	// Name 返回触发该命令的名称（不含前导斜杠，如 "new"）。
	Name() string
	// Description 返回该命令的简短描述。
	Description() string
	// Execute 执行命令逻辑。args 是命令名称后紧跟的参数文本。
	Execute(ctx context.Context, args string, env Env) Result
}

// Registry 负责存储所有已注册的斜杠命令并进行匹配分发。
type Registry struct {
	mu   sync.RWMutex
	cmds map[string]Command
}

// NewRegistry 创建并返回一个空的命令注册表。
func NewRegistry() *Registry {
	return &Registry{cmds: make(map[string]Command)}
}

// Register 向注册表中添加一个命令。如果存在同名命令，将触发 panic。
func (r *Registry) Register(cmd Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := strings.ToLower(cmd.Name())
	if _, dup := r.cmds[name]; dup {
		panic("command already registered: " + name)
	}
	r.cmds[name] = cmd
}

// Lookup 解析用户输入。如果输入以 "/" 开头且匹配已注册命令，则返回该命令及其参数。
func (r *Registry) Lookup(content string) (Command, string, bool) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "/") {
		return nil, "", false
	}
	body := content[1:]
	name, args, _ := strings.Cut(body, " ")
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, "", false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.cmds[name]
	if !ok {
		return nil, "", false
	}
	return cmd, strings.TrimSpace(args), true
}

// List 返回按名称字母顺序排序的所有已注册命令列表。
func (r *Registry) List() []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Command, 0, len(r.cmds))
	for _, cmd := range r.cmds {
		out = append(out, cmd)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
