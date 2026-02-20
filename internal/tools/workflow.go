package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// WorkflowRequest describes a structured subagent workflow run.
type WorkflowRequest struct {
	Goal           string
	Mode           string
	Subtasks       []string
	Label          string
	OriginChannel  string
	OriginChatID   string
	OriginSenderID string
	RequestID      string
}

// WorkflowExecutor executes delegated subagent workflows.
type WorkflowExecutor interface {
	RunWorkflow(ctx context.Context, req WorkflowRequest) (string, error)
}

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

// NewWorkflowTool creates a workflow orchestration tool.
func NewWorkflowTool(executor WorkflowExecutor) (tool.InvokableTool, error) {
	impl := &workflowToolImpl{executor: executor}
	return utils.InferTool(
		"workflow",
		"Split a goal into subtasks, execute via subagents in sequential/parallel mode, and return aggregated results.",
		impl.execute,
	)
}
