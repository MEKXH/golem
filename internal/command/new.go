package command

import (
	"context"
	"log/slog"
)

// NewSessionCommand 实现 /new 命令 — 用于重置当前会话，清除历史上下文。
type NewSessionCommand struct{}

// Name 返回命令名称。
func (c *NewSessionCommand) Name() string { return "new" }

// Description 返回命令描述。
func (c *NewSessionCommand) Description() string { return "Start a new conversation session" }

// Execute 执行重置会话逻辑。
func (c *NewSessionCommand) Execute(_ context.Context, _ string, env Env) Result {
	env.Sessions.Reset(env.SessionKey)
	slog.Info("session reset via /new", "session_key", env.SessionKey, "channel", env.Channel, "chat_id", env.ChatID)
	return Result{Content: "New session started."}
}
