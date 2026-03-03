package command

import (
	"context"
	"fmt"
	"strings"
)

// HelpCommand 实现 /help 命令 — 用于列出当前所有已注册且可用的斜杠命令。
type HelpCommand struct{}

// Name 返回命令名称。
func (c *HelpCommand) Name() string        { return "help" }

// Description 返回命令描述。
func (c *HelpCommand) Description() string { return "List available slash commands" }

// Execute 执行列出命令的逻辑。
func (c *HelpCommand) Execute(_ context.Context, _ string, env Env) Result {
	cmds := env.ListCommands()
	var sb strings.Builder
	sb.WriteString("**Available commands:**\n\n")
	for _, cmd := range cmds {
		sb.WriteString(fmt.Sprintf("- `/%s` — %s\n", cmd.Name(), cmd.Description()))
	}
	return Result{Content: sb.String()}
}
