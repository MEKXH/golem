package agent

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/approval"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type policyE2EModel struct {
	toolName string
	argsJSON string
}

func (m *policyE2EModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	var lastToolResult string
	for _, msg := range input {
		if msg.Role == schema.Tool {
			lastToolResult = msg.Content
		}
	}

	if lastToolResult == "" {
		return &schema.Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					ID: "policy-e2e-call",
					Function: schema.FunctionCall{
						Name:      m.toolName,
						Arguments: m.argsJSON,
					},
				},
			},
		}, nil
	}

	return &schema.Message{
		Role:    schema.Assistant,
		Content: lastToolResult,
	}, nil
}

func (m *policyE2EModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *policyE2EModel) BindTools(toolInfos []*schema.ToolInfo) error {
	return nil
}

func TestE2E_StrictMode_BlocksExecWithoutApproval(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	cfg.Policy.Mode = "strict"
	cfg.Policy.RequireApproval = []string{"exec"}
	cfg.Tools.Exec.RestrictToWorkspace = false

	msgBus := bus.NewMessageBus(1)
	loop, err := NewLoop(cfg, msgBus, &policyE2EModel{
		toolName: "exec",
		argsJSON: `{"command":"echo strict-blocked"}`,
	})
	if err != nil {
		t.Fatalf("NewLoop() error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools() error: %v", err)
	}

	resp, err := loop.ProcessDirect(context.Background(), "trigger strict policy")
	if err != nil {
		t.Fatalf("ProcessDirect() error: %v", err)
	}
	if !strings.Contains(resp, "approval required") {
		t.Fatalf("expected approval path response, got: %s", resp)
	}
}

func TestE2E_StrictMode_ApprovedRequestExecutes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	cfg.Policy.Mode = "strict"
	cfg.Policy.RequireApproval = []string{"exec"}
	cfg.Tools.Exec.RestrictToWorkspace = false

	const argsJSON = `{"command":"echo approved-e2e"}`

	msgBus := bus.NewMessageBus(1)
	loop, err := NewLoop(cfg, msgBus, &policyE2EModel{
		toolName: "exec",
		argsJSON: argsJSON,
	})
	if err != nil {
		t.Fatalf("NewLoop() error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools() error: %v", err)
	}

	approvalSvc := approval.NewService(loop.workspacePath)
	req, err := approvalSvc.Create(approval.CreateInput{
		ToolName: "exec",
		ArgsJSON: normalizeArgsJSON(argsJSON),
		Reason:   "test approved flow",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := approvalSvc.Approve(req.ID, approval.DecisionInput{DecidedBy: "tester"}); err != nil {
		t.Fatalf("Approve() error: %v", err)
	}

	resp, err := loop.ProcessDirect(context.Background(), "trigger approved execution")
	if err != nil {
		t.Fatalf("ProcessDirect() error: %v", err)
	}
	if strings.Contains(resp, "approval required") {
		t.Fatalf("expected approved execution to proceed, got: %s", resp)
	}
	if !strings.Contains(strings.ToLower(resp), "approved-e2e") {
		t.Fatalf("expected command output in response, got: %s", resp)
	}
}

func TestE2E_OffModeWithTTL_RevertsToStrict(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = "100ms"
	cfg.Policy.RequireApproval = []string{"exec"}
	cfg.Tools.Exec.RestrictToWorkspace = false

	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	msgBus := bus.NewMessageBus(1)
	loop, err := NewLoop(cfg, msgBus, nil)
	if err != nil {
		t.Fatalf("NewLoop() error: %v", err)
	}
	loop.now = func() time.Time { return now }
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools() error: %v", err)
	}

	firstResult, err := loop.tools.Execute(context.Background(), "exec", `{"command":"echo ttl-open"}`)
	if err != nil {
		t.Fatalf("first Execute() error: %v", err)
	}
	if strings.Contains(firstResult, "approval required") {
		t.Fatalf("expected off mode to allow execution before ttl expires, got: %s", firstResult)
	}

	now = now.Add(2 * time.Second)
	secondResult, err := loop.tools.Execute(context.Background(), "exec", `{"command":"echo ttl-open"}`)
	if err != nil {
		t.Fatalf("second Execute() error: %v", err)
	}
	if !strings.Contains(secondResult, "approval required") {
		t.Fatalf("expected strict fallback after ttl expiry, got: %s", secondResult)
	}
}

func TestE2E_MCPUnavailableServer_DoesNotCrashLoop(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	cfg.MCP.Servers = map[string]config.MCPServerConfig{
		"broken": {
			Transport: "stdio",
			Command:   "missing-mcp-server",
		},
	}

	msgBus := bus.NewMessageBus(1)
	loop, err := NewLoop(cfg, msgBus, nil)
	if err != nil {
		t.Fatalf("NewLoop() error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools() should not fail on degraded MCP server, got: %v", err)
	}
	if loop.mcpManager == nil {
		t.Fatal("expected mcp manager to be initialized")
	}

	statuses := loop.mcpManager.Statuses()
	if len(statuses) != 1 {
		t.Fatalf("expected one server status, got %d", len(statuses))
	}
	if !statuses[0].Degraded {
		t.Fatalf("expected degraded status for unavailable server, got: %+v", statuses[0])
	}

	names := loop.tools.Names()
	hasMCPTool := slices.ContainsFunc(names, func(name string) bool {
		return strings.HasPrefix(name, "mcp.")
	})
	if hasMCPTool {
		t.Fatalf("did not expect MCP tools to be registered for degraded server, got: %v", names)
	}

	resp, err := loop.ProcessDirect(context.Background(), "hello")
	if err != nil {
		t.Fatalf("ProcessDirect() error: %v", err)
	}
	if resp != "No model configured" {
		t.Fatalf("expected loop to stay operational without model, got: %s", resp)
	}
}
