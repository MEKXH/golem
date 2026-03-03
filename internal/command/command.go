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

// Env 携带斜杠命令的每次调用上下文。
type Env struct {
	Channel       string
	ChatID        string
	SenderID      string
	SessionKey    string
	Sessions      *session.Manager
	WorkspacePath string
	Config        *config.Config
	Metrics       *metrics.RuntimeMetrics
	ListCommands  func() []Command // for /help
}

// Result 是斜杠命令执行的输出。
type Result struct {
	Content string
}

// Command 是每个斜杠命令必须实现的接口。
type Command interface {
	// Name returns the command trigger without the leading slash (e.g. "new").
	Name() string
	// Description returns a short human-readable summary.
	Description() string
	// Execute runs the command. args is the trimmed text after the command name.
	Execute(ctx context.Context, args string, env Env) Result
}

// Registry 存储已注册的斜杠命令并分发执行。
type Registry struct {
	mu   sync.RWMutex
	cmds map[string]Command
}

// NewRegistry 创建一个空的命令注册表。
func NewRegistry() *Registry {
	return &Registry{cmds: make(map[string]Command)}
}

// Register 添加一个命令。重复名称时 panic。
func (r *Registry) Register(cmd Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := strings.ToLower(cmd.Name())
	if _, dup := r.cmds[name]; dup {
		panic("command already registered: " + name)
	}
	r.cmds[name] = cmd
}

// Lookup 解析原始用户输入。如果以 "/" 开头且匹配已注册的命令，
// 则返回命令、剩余参数和 true。
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

// List 返回按名称排序的所有已注册命令。
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
