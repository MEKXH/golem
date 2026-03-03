package command

import (
	"context"
	"fmt"
	"strings"
)

// HelpCommand 实现 /help — 列出所有可用的斜杠命令。
type HelpCommand struct{}

func (c *HelpCommand) Name() string        { return "help" }
func (c *HelpCommand) Description() string { return "List available slash commands" }

func (c *HelpCommand) Execute(_ context.Context, _ string, env Env) Result {
	cmds := env.ListCommands()
	var sb strings.Builder
	sb.WriteString("**Available commands:**\n\n")
	for _, cmd := range cmds {
		sb.WriteString(fmt.Sprintf("- `/%s` — %s\n", cmd.Name(), cmd.Description()))
	}
	return Result{Content: sb.String()}
}
