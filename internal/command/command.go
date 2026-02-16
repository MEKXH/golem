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

// Env carries per-invocation context for a slash command.
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

// Result is the output of a slash command execution.
type Result struct {
	Content string
}

// Command is the interface every slash command must implement.
type Command interface {
	// Name returns the command trigger without the leading slash (e.g. "new").
	Name() string
	// Description returns a short human-readable summary.
	Description() string
	// Execute runs the command. args is the trimmed text after the command name.
	Execute(ctx context.Context, args string, env Env) Result
}

// Registry holds registered slash commands and dispatches them.
type Registry struct {
	mu   sync.RWMutex
	cmds map[string]Command
}

// NewRegistry creates an empty command registry.
func NewRegistry() *Registry {
	return &Registry{cmds: make(map[string]Command)}
}

// Register adds a command. Panics on duplicate names.
func (r *Registry) Register(cmd Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := strings.ToLower(cmd.Name())
	if _, dup := r.cmds[name]; dup {
		panic("command already registered: " + name)
	}
	r.cmds[name] = cmd
}

// Lookup parses raw user input. If it starts with "/" and matches a registered
// command, it returns the command, the remaining args, and true.
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

// List returns all registered commands sorted by name.
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
