package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
	subagents     *SubagentManager
	sessions      *session.Manager
	context       *ContextBuilder
	maxIterations int
	workspacePath string

	OnToolStart  func(name, args string)
	OnToolFinish func(name, result string, err error)

	activityRecorder func(channel, chatID string)
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

// Tools returns the tool registry.
func (l *Loop) Tools() *tools.Registry {
	return l.tools
}

// SetActivityRecorder attaches a callback used to track the latest active channel/chat.
func (l *Loop) SetActivityRecorder(recorder func(channel, chatID string)) {
	l.activityRecorder = recorder
}

// RegisterDefaultTools registers all built-in tools
func (l *Loop) RegisterDefaultTools(cfg *config.Config) error {
	toolFns := []func() (tool.InvokableTool, error){
		func() (tool.InvokableTool, error) { return tools.NewReadFileTool(l.workspacePath) },
		func() (tool.InvokableTool, error) { return tools.NewWriteFileTool(l.workspacePath) },
		func() (tool.InvokableTool, error) { return tools.NewListDirTool(l.workspacePath) },
		func() (tool.InvokableTool, error) { return tools.NewReadMemoryTool(l.workspacePath) },
		func() (tool.InvokableTool, error) { return tools.NewWriteMemoryTool(l.workspacePath) },
		func() (tool.InvokableTool, error) { return tools.NewAppendDiaryTool(l.workspacePath) },
		func() (tool.InvokableTool, error) {
			return tools.NewExecTool(
				cfg.Tools.Exec.Timeout,
				cfg.Tools.Exec.RestrictToWorkspace,
				l.workspacePath,
			)
		},
		func() (tool.InvokableTool, error) { return tools.NewWebFetchTool() },
		func() (tool.InvokableTool, error) {
			return tools.NewWebSearchTool(cfg.Tools.Web.Search.APIKey, cfg.Tools.Web.Search.MaxResults)
		},
	}

	registered := make([]string, 0, len(toolFns))
	for _, fn := range toolFns {
		t, err := fn()
		if err != nil {
			return err
		}
		if err := l.tools.Register(t); err != nil {
			return err
		}
		info, err := t.Info(context.Background())
		if err == nil && info != nil && info.Name != "" {
			registered = append(registered, info.Name)
		}
	}

	msgTool, err := tools.NewMessageTool(l.bus)
	if err != nil {
		return err
	}
	if err := l.tools.Register(msgTool); err != nil {
		return err
	}
	if info, err := msgTool.Info(context.Background()); err == nil && info != nil && info.Name != "" {
		registered = append(registered, info.Name)
	}

	l.subagents = NewSubagentManager(l.bus, l, 5*time.Minute)
	spawnTool, err := tools.NewSpawnTool(l.subagents)
	if err != nil {
		return err
	}
	if err := l.tools.Register(spawnTool); err != nil {
		return err
	}
	if info, err := spawnTool.Info(context.Background()); err == nil && info != nil && info.Name != "" {
		registered = append(registered, info.Name)
	}

	subagentTool, err := tools.NewSubagentTool(l.subagents)
	if err != nil {
		return err
	}
	if err := l.tools.Register(subagentTool); err != nil {
		return err
	}
	if info, err := subagentTool.Info(context.Background()); err == nil && info != nil && info.Name != "" {
		registered = append(registered, info.Name)
	}

	slog.Info("registered tools", "count", len(registered), "tools", registered)
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
		case msg, ok := <-l.bus.Inbound():
			if !ok {
				return fmt.Errorf("inbound channel closed")
			}
			if msg == nil {
				slog.Warn("received nil inbound message")
				continue
			}
			if strings.TrimSpace(msg.RequestID) == "" {
				msg.RequestID = bus.NewRequestID()
			}
			if msg.Channel == bus.SystemChannel {
				l.processSystemMessage(msg)
				continue
			}
			resp, err := l.processMessage(ctx, msg)
			if err != nil {
				slog.Error("process message failed", "request_id", msg.RequestID, "channel", msg.Channel, "chat_id", msg.ChatID, "session_key", msg.SessionKey(), "error", err)
				l.bus.PublishOutbound(&bus.OutboundMessage{
					Channel:   msg.Channel,
					ChatID:    msg.ChatID,
					Content:   "Error: " + err.Error(),
					RequestID: msg.RequestID,
				})
				continue
			}
			if resp != nil {
				l.bus.PublishOutbound(resp)
			}
		}
	}
}

func (l *Loop) processSystemMessage(msg *bus.InboundMessage) {
	if msg == nil {
		return
	}

	msgType := strings.TrimSpace(fmt.Sprint(msg.Metadata[bus.SystemMetaType]))
	if msgType != bus.SystemTypeSubagentResult {
		slog.Info("ignored system message", "request_id", msg.RequestID, "type", msgType)
		return
	}

	originChannel := strings.TrimSpace(fmt.Sprint(msg.Metadata[bus.SystemMetaOriginChannel]))
	if originChannel == "" {
		originChannel = "cli"
	}
	originChatID := strings.TrimSpace(fmt.Sprint(msg.Metadata[bus.SystemMetaOriginChatID]))
	if originChatID == "" {
		originChatID = "direct"
	}

	label := strings.TrimSpace(fmt.Sprint(msg.Metadata[bus.SystemMetaTaskLabel]))
	content := strings.TrimSpace(msg.Content)
	if label != "" {
		content = fmt.Sprintf("Subagent '%s' completed.\n\n%s", label, content)
	}
	if content == "" {
		content = "Subagent completed."
	}

	l.bus.PublishOutbound(&bus.OutboundMessage{
		Channel:   originChannel,
		ChatID:    originChatID,
		Content:   content,
		RequestID: msg.RequestID,
		Metadata: map[string]any{
			bus.SystemMetaType:   bus.SystemTypeSubagentResult,
			bus.SystemMetaTaskID: msg.Metadata[bus.SystemMetaTaskID],
			bus.SystemMetaStatus: msg.Metadata[bus.SystemMetaStatus],
		},
	})
}

func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) (*bus.OutboundMessage, error) {
	slog.Info("processing message", "request_id", msg.RequestID, "channel", msg.Channel, "chat_id", msg.ChatID, "sender", msg.SenderID, "session_key", msg.SessionKey())
	if l.activityRecorder != nil {
		l.activityRecorder(msg.Channel, msg.ChatID)
	}

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
			toolStart := time.Now()
			slog.Debug("executing tool", "request_id", msg.RequestID, "name", tc.Function.Name)

			if l.OnToolStart != nil {
				l.OnToolStart(tc.Function.Name, tc.Function.Arguments)
			}

			toolCtx := tools.WithInvocationContext(ctx, tools.InvocationContext{
				Channel:   msg.Channel,
				ChatID:    msg.ChatID,
				SenderID:  msg.SenderID,
				RequestID: msg.RequestID,
				SessionID: msg.SessionKey(),
			})

			result, err := l.tools.Execute(toolCtx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = "Error: " + err.Error()
			}
			toolDuration := time.Since(toolStart)
			slog.Info("tool execution finished",
				"request_id", msg.RequestID,
				"channel", msg.Channel,
				"chat_id", msg.ChatID,
				"tool", tc.Function.Name,
				"tool_duration", toolDuration.String(),
				"duration_ms", toolDuration.Milliseconds(),
				"success", err == nil,
			)

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
		Channel:   msg.Channel,
		ChatID:    msg.ChatID,
		Content:   finalContent,
		RequestID: msg.RequestID,
	}, nil
}

// ProcessForChannel processes a message directly for a given channel/session.
func (l *Loop) ProcessForChannel(ctx context.Context, channel, chatID, senderID, content string) (string, error) {
	return l.ProcessForChannelWithSession(ctx, channel, chatID, senderID, "", content)
}

// ProcessForChannelWithSession processes a message for a channel/chat using an optional explicit session id.
func (l *Loop) ProcessForChannelWithSession(ctx context.Context, channel, chatID, senderID, sessionID, content string) (string, error) {
	if err := l.bindTools(ctx); err != nil {
		return "", err
	}
	if strings.TrimSpace(channel) == "" {
		channel = "cli"
	}
	if strings.TrimSpace(chatID) == "" {
		chatID = "direct"
	}
	if strings.TrimSpace(senderID) == "" {
		senderID = "user"
	}

	msg := &bus.InboundMessage{
		Channel:   channel,
		SenderID:  senderID,
		ChatID:    chatID,
		SessionID: strings.TrimSpace(sessionID),
		Content:   content,
		RequestID: bus.RequestIDFromContext(ctx),
	}
	if strings.TrimSpace(msg.RequestID) == "" {
		msg.RequestID = bus.NewRequestID()
	}

	resp, err := l.processMessage(ctx, msg)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// ProcessDirect processes a message directly (for CLI)
func (l *Loop) ProcessDirect(ctx context.Context, content string) (string, error) {
	return l.ProcessForChannel(ctx, "cli", "direct", "user", content)
}
