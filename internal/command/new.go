package command

import (
	"context"
	"log/slog"
)

// NewSessionCommand implements /new — resets the current conversation session.
type NewSessionCommand struct{}

func (c *NewSessionCommand) Name() string        { return "new" }
func (c *NewSessionCommand) Description() string { return "Start a new conversation session" }

func (c *NewSessionCommand) Execute(_ context.Context, _ string, env Env) Result {
	env.Sessions.Reset(env.SessionKey)
	slog.Info("session reset via /new", "session_key", env.SessionKey, "channel", env.Channel, "chat_id", env.ChatID)
	return Result{Content: "New session started."}
}
