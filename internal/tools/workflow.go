package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// WorkflowRequest 描述结构化的子代理工作流运行请求。
type WorkflowRequest struct {
	Goal           string   // 整个工作流的最终目标
	Mode           string   // 执行模式：sequential (顺序) 或 parallel (并行)
	Subtasks       []string // 预定义的子任务列表
	Label          string   // 用于追踪的工作流标签
	OriginChannel  string   // 原始请求通道
	OriginChatID   string   // 原始聊天 ID
	OriginSenderID string   // 原始发送者 ID
	RequestID      string   // 请求追踪 ID
}

// WorkflowExecutor 定义了执行子代理工作流的接口。
type WorkflowExecutor interface {
	// RunWorkflow 执行委托的工作流并返回汇总结果。
	RunWorkflow(ctx context.Context, req WorkflowRequest) (string, error)
}

// WorkflowInput 定义了 workflow 工具的输入参数。
type WorkflowInput struct {
	Goal     string   `json:"goal" jsonschema:"required,description=Overall workflow goal for decomposition and execution"`
	Mode     string   `json:"mode,omitempty" jsonschema:"description=Execution mode: sequential or parallel"`
	Subtasks []string `json:"subtasks,omitempty" jsonschema:"description=Optional predefined subtasks"`
	Label    string   `json:"label,omitempty" jsonschema:"description=Optional workflow label for tracking"`
}

type workflowToolImpl struct {
	executor WorkflowExecutor
}

func (t *workflowToolImpl) execute(ctx context.Context, input *WorkflowInput) (string, error) {
	req, err := buildWorkflowRequest(ctx, input)
	if err != nil {
		return "", err
	}
	if t.executor == nil {
		return "", fmt.Errorf("workflow executor is not configured")
	}
	return t.executor.RunWorkflow(ctx, req)
}

func buildWorkflowRequest(ctx context.Context, input *WorkflowInput) (WorkflowRequest, error) {
	if input == nil {
		return WorkflowRequest{}, fmt.Errorf("workflow input is required")
	}
	goal := strings.TrimSpace(input.Goal)
	if goal == "" {
		return WorkflowRequest{}, fmt.Errorf("goal is required")
	}

	meta := InvocationFromContext(ctx)
	channel := strings.TrimSpace(meta.Channel)
	if channel == "" {
		channel = "cli"
	}
	chatID := strings.TrimSpace(meta.ChatID)
	if chatID == "" {
		chatID = "direct"
	}
	senderID := strings.TrimSpace(meta.SenderID)
	if senderID == "" {
		senderID = "user"
	}

	subtasks := make([]string, 0, len(input.Subtasks))
	for _, raw := range input.Subtasks {
		task := strings.TrimSpace(raw)
		if task != "" {
			subtasks = append(subtasks, task)
		}
	}

	return WorkflowRequest{
		Goal:           goal,
		Mode:           strings.ToLower(strings.TrimSpace(input.Mode)),
		Subtasks:       subtasks,
		Label:          strings.TrimSpace(input.Label),
		OriginChannel:  channel,
		OriginChatID:   chatID,
		OriginSenderID: senderID,
		RequestID:      strings.TrimSpace(meta.RequestID),
	}, nil
}

// NewWorkflowTool 创建一个工作流编排工具实例，用于将目标分解为子任务并委派执行。
func NewWorkflowTool(executor WorkflowExecutor) (tool.InvokableTool, error) {
	impl := &workflowToolImpl{executor: executor}
	return utils.InferTool(
		"workflow",
		"Split a goal into subtasks, execute via subagents in sequential/parallel mode, and return aggregated results.",
		impl.execute,
	)
}
