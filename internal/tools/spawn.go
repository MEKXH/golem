package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// SubagentRequest 封装了子代理工具执行所需的规范化请求信息。
type SubagentRequest struct {
	Task           string // 需要子代理执行的任务描述
	Label          string // 任务的可选描述性标签
	OriginChannel  string // 原始请求通道
	OriginChatID   string // 原始聊天 ID
	OriginSenderID string // 原始发送者 ID
	RequestID      string // 请求追踪 ID
}

// SubagentExecutor 定义了派生或同步运行子代理任务的接口。
type SubagentExecutor interface {
	// Spawn 异步启动任务。
	Spawn(ctx context.Context, req SubagentRequest) (string, error)
	// RunSync 同步运行任务并等待结果。
	RunSync(ctx context.Context, req SubagentRequest) (string, error)
}

// SpawnInput 定义了 spawn 工具（异步委派）的输入参数。
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

// SubagentInput 定义了 subagent 工具（同步委派）的输入参数。
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

	// 优先使用参数中指定的通道/聊天 ID，否则从上下文中继承
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

// NewSpawnTool 创建一个异步子代理委托工具，任务在后台运行并通过异步回调上报完成。
func NewSpawnTool(executor SubagentExecutor) (tool.InvokableTool, error) {
	impl := &spawnToolImpl{executor: executor}
	return utils.InferTool(
		"spawn",
		"Run a delegated subagent task in background and report completion asynchronously.",
		impl.execute,
	)
}

// NewSubagentTool 创建一个同步子代理委托工具，任务同步运行并直接返回执行结果。
func NewSubagentTool(executor SubagentExecutor) (tool.InvokableTool, error) {
	impl := &subagentToolImpl{executor: executor}
	return utils.InferTool(
		"subagent",
		"Run a delegated subagent task synchronously and return the result.",
		impl.execute,
	)
}
