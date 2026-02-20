package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// SubagentRequest is the normalized execution request for subagent tools.
type SubagentRequest struct {
	Task           string
	Label          string
	OriginChannel  string
	OriginChatID   string
	OriginSenderID string
	RequestID      string
}

// SubagentExecutor executes delegated subagent tasks.
type SubagentExecutor interface {
	Spawn(ctx context.Context, req SubagentRequest) (string, error)
	RunSync(ctx context.Context, req SubagentRequest) (string, error)
}

type SpawnInput struct {
	Task    string `json:"task" jsonschema:"required,description=Task to delegate to a background subagent"`
	Label   string `json:"label,omitempty" jsonschema:"description=Optional label for task tracking"`
	Channel string `json:"channel,omitempty" jsonschema:"description=Optional origin channel override"`
	ChatID  string `json:"chat_id,omitempty" jsonschema:"description=Optional origin chat id override"`
}

type spawnToolImpl struct {
	executor SubagentExecutor
}

func (t *spawnToolImpl) execute(ctx context.Context, input *SpawnInput) (string, error) {
	req, err := buildSubagentRequest(ctx, input.Task, input.Label, input.Channel, input.ChatID)
	if err != nil {
		return "", err
	}
	if t.executor == nil {
		return "", fmt.Errorf("subagent executor is not configured")
	}
	taskID, err := t.executor.Spawn(ctx, req)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Subagent task started: %s", taskID), nil
}

type SubagentInput struct {
	Task    string `json:"task" jsonschema:"required,description=Task to execute via a delegated subagent"`
	Label   string `json:"label,omitempty" jsonschema:"description=Optional label for this delegated run"`
	Channel string `json:"channel,omitempty" jsonschema:"description=Optional origin channel override"`
	ChatID  string `json:"chat_id,omitempty" jsonschema:"description=Optional origin chat id override"`
}

type subagentToolImpl struct {
	executor SubagentExecutor
}

func (t *subagentToolImpl) execute(ctx context.Context, input *SubagentInput) (string, error) {
	req, err := buildSubagentRequest(ctx, input.Task, input.Label, input.Channel, input.ChatID)
	if err != nil {
		return "", err
	}
	if t.executor == nil {
		return "", fmt.Errorf("subagent executor is not configured")
	}
	return t.executor.RunSync(ctx, req)
}

func buildSubagentRequest(ctx context.Context, task, label, channel, chatID string) (SubagentRequest, error) {
	task = strings.TrimSpace(task)
	if task == "" {
		return SubagentRequest{}, fmt.Errorf("task is required")
	}
	meta := InvocationFromContext(ctx)

	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = meta.Channel
	}
	if channel == "" {
		channel = "cli"
	}

	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		chatID = meta.ChatID
	}
	if chatID == "" {
		chatID = "direct"
	}

	sender := meta.SenderID
	if sender == "" {
		sender = "user"
	}

	return SubagentRequest{
		Task:           task,
		Label:          strings.TrimSpace(label),
		OriginChannel:  channel,
		OriginChatID:   chatID,
		OriginSenderID: sender,
		RequestID:      meta.RequestID,
	}, nil
}

// NewSpawnTool creates an async subagent delegation tool.
func NewSpawnTool(executor SubagentExecutor) (tool.InvokableTool, error) {
	impl := &spawnToolImpl{executor: executor}
	return utils.InferTool(
		"spawn",
		"Run a delegated subagent task in background and report completion asynchronously.",
		impl.execute,
	)
}

// NewSubagentTool creates a sync subagent delegation tool.
func NewSubagentTool(executor SubagentExecutor) (tool.InvokableTool, error) {
	impl := &subagentToolImpl{executor: executor}
	return utils.InferTool(
		"subagent",
		"Run a delegated subagent task synchronously and return the result.",
		impl.execute,
	)
}
