package agent

import (
	"context"
	"log/slog"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/session"
	"github.com/MEKXH/golem/internal/tools"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Loop is the main agent processing loop
type Loop struct {
	bus           *bus.MessageBus
	model         model.ChatModel
	tools         *tools.Registry
	sessions      *session.Manager
	context       *ContextBuilder
	maxIterations int
	workspacePath string

	OnToolStart  func(name, args string)
	OnToolFinish func(name, result string, err error)
}

// NewLoop creates a new agent loop
func NewLoop(cfg *config.Config, msgBus *bus.MessageBus, chatModel model.ChatModel) (*Loop, error) {
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return nil, err
	}
	return &Loop{
		bus:           msgBus,
		model:         chatModel,
		tools:         tools.NewRegistry(),
		sessions:      session.NewManager(workspacePath),
		context:       NewContextBuilder(workspacePath),
		maxIterations: cfg.Agents.Defaults.MaxToolIterations,
		workspacePath: workspacePath,
	}, nil
}

// RegisterDefaultTools registers all built-in tools
func (l *Loop) RegisterDefaultTools(cfg *config.Config) error {
	toolFns := []func() (interface{}, error){
		func() (interface{}, error) { return tools.NewReadFileTool(l.workspacePath) },
		func() (interface{}, error) { return tools.NewWriteFileTool(l.workspacePath) },
		func() (interface{}, error) { return tools.NewListDirTool(l.workspacePath) },
		func() (interface{}, error) {
			return tools.NewExecTool(
				cfg.Tools.Exec.Timeout,
				cfg.Tools.Exec.RestrictToWorkspace,
				l.workspacePath,
			)
		},
	}

	for _, fn := range toolFns {
		t, err := fn()
		if err != nil {
			return err
		}
		if invokable, ok := t.(tool.InvokableTool); ok {
			if err := l.tools.Register(invokable); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Loop) bindTools(ctx context.Context) error {
	if l.model == nil {
		return nil
	}
	toolInfos, err := l.tools.GetToolInfos(ctx)
	if err != nil {
		return err
	}
	if binder, ok := l.model.(interface {
		BindTools([]*schema.ToolInfo) error
	}); ok {
		return binder.BindTools(toolInfos)
	}
	return nil
}

// Run starts the agent loop
func (l *Loop) Run(ctx context.Context) error {
	if err := l.bindTools(ctx); err != nil {
		return err
	}

	slog.Info("agent loop started")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-l.bus.Inbound():
			resp, err := l.processMessage(ctx, msg)
			if err != nil {
				slog.Error("process message failed", "error", err)
				l.bus.PublishOutbound(&bus.OutboundMessage{
					Channel: msg.Channel,
					ChatID:  msg.ChatID,
					Content: "Error: " + err.Error(),
				})
				continue
			}
			if resp != nil {
				l.bus.PublishOutbound(resp)
			}
		}
	}
}

func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) (*bus.OutboundMessage, error) {
	slog.Info("processing message", "channel", msg.Channel, "sender", msg.SenderID)

	sess := l.sessions.GetOrCreate(msg.SessionKey())

	messages := l.context.BuildMessages(sess.GetHistory(50), msg.Content, msg.Media)

	var finalContent string

	for i := 0; i < l.maxIterations; i++ {
		if l.model == nil {
			finalContent = "No model configured"
			break
		}

		resp, err := l.model.Generate(ctx, messages)
		if err != nil {
			return nil, err
		}

		if len(resp.ToolCalls) == 0 {
			finalContent = resp.Content
			break
		}

		messages = append(messages, resp)

		for _, tc := range resp.ToolCalls {
			slog.Debug("executing tool", "name", tc.Function.Name)

			if l.OnToolStart != nil {
				l.OnToolStart(tc.Function.Name, tc.Function.Arguments)
			}

			result, err := l.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = "Error: " + err.Error()
			}

			if l.OnToolFinish != nil {
				l.OnToolFinish(tc.Function.Name, result, err)
			}

			messages = append(messages, &schema.Message{
				Role:       schema.Tool,
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	if finalContent == "" {
		finalContent = "Processing complete."
	}

	sess.AddMessage("user", msg.Content)
	sess.AddMessage("assistant", finalContent)
	l.sessions.Save(sess)

	return &bus.OutboundMessage{
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Content: finalContent,
	}, nil
}

// ProcessDirect processes a message directly (for CLI)
func (l *Loop) ProcessDirect(ctx context.Context, content string) (string, error) {
	if err := l.bindTools(ctx); err != nil {
		return "", err
	}
	msg := &bus.InboundMessage{
		Channel:  "cli",
		SenderID: "user",
		ChatID:   "direct",
		Content:  content,
	}

	resp, err := l.processMessage(ctx, msg)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
