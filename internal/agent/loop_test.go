package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/session"
	"github.com/MEKXH/golem/internal/tools"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type mockChatModel struct {
	bindCalls  int
	boundTools int
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return &schema.Message{Role: schema.Assistant, Content: "ok"}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	m.bindCalls++
	m.boundTools = len(tools)
	return nil
}

func TestNewLoop(t *testing.T) {
	cfg := config.DefaultConfig()
	msgBus := bus.NewMessageBus(10)

	loop, err := NewLoop(cfg, msgBus, nil)
	if err != nil {
		t.Fatalf("NewLoop error: %v", err)
	}
	if loop == nil {
		t.Fatal("expected non-nil Loop")
	}
	if loop.maxIterations != 20 {
		t.Errorf("expected maxIterations=20, got %d", loop.maxIterations)
	}
}

func TestNewLoop_InvalidWorkspaceMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.WorkspaceMode = "path"
	cfg.Agents.Defaults.Workspace = ""
	msgBus := bus.NewMessageBus(10)

	if _, err := NewLoop(cfg, msgBus, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestContextBuilder_BuildSystemPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	cb := NewContextBuilder(tmpDir)

	prompt := cb.BuildSystemPrompt()
	if !strings.Contains(prompt, "Golem") {
		t.Error("expected system prompt to contain 'Golem'")
	}
}

func TestProcessDirect_BindsTools(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	msgBus := bus.NewMessageBus(1)
	model := &mockChatModel{}

	loop, err := NewLoop(cfg, msgBus, model)
	if err != nil {
		t.Fatalf("NewLoop error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools error: %v", err)
	}

	if got := len(loop.tools.Names()); got == 0 {
		t.Fatalf("expected tools registered, got %d", got)
	}

	_, err = loop.ProcessDirect(context.Background(), "hi")
	if err != nil {
		t.Fatalf("ProcessDirect error: %v", err)
	}

	if model.bindCalls == 0 {
		t.Fatalf("expected BindTools to be called")
	}
	if model.boundTools == 0 {
		t.Fatalf("expected tools to be bound")
	}
}

// multiTurnMockModel returns a tool call on the first Generate call, then a final text response.
type multiTurnMockModel struct {
	callCount int
}

func (m *multiTurnMockModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.callCount++
	if m.callCount == 1 {
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "",
			ToolCalls: []schema.ToolCall{
				{
					ID: "call_1",
					Function: schema.FunctionCall{
						Name:      "mock_tool",
						Arguments: `{"input":"test"}`,
					},
				},
			},
		}, nil
	}
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "Final response",
	}, nil
}

func (m *multiTurnMockModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *multiTurnMockModel) BindTools(toolInfos []*schema.ToolInfo) error {
	return nil
}

type geoPipelineMockModel struct {
	callCount int
}

func (m *geoPipelineMockModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.callCount++
	if m.callCount == 1 {
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "",
			ToolCalls: []schema.ToolCall{
				{
					ID: "geo_call_1",
					Function: schema.FunctionCall{
						Name:      "geo_info",
						Arguments: `{"path":"river.geojson"}`,
					},
				},
				{
					ID: "geo_call_2",
					Function: schema.FunctionCall{
						Name:      "geo_sinuosity",
						Arguments: `{"input_path":"river.geojson"}`,
					},
				},
			},
		}, nil
	}
	return &schema.Message{Role: schema.Assistant, Content: "Pipeline complete"}, nil
}

func (m *geoPipelineMockModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *geoPipelineMockModel) BindTools(toolInfos []*schema.ToolInfo) error {
	return nil
}

// alwaysToolCallModel always returns a tool call, never a final response.
type alwaysToolCallModel struct {
	callCount int
}

func (m *alwaysToolCallModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.callCount++
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "",
		ToolCalls: []schema.ToolCall{
			{
				ID: "call_" + fmt.Sprintf("%d", m.callCount),
				Function: schema.FunctionCall{
					Name:      "mock_tool",
					Arguments: `{"input":"loop"}`,
				},
			},
		},
	}, nil
}

func (m *alwaysToolCallModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *alwaysToolCallModel) BindTools(toolInfos []*schema.ToolInfo) error {
	return nil
}

// testTool is a simple mock tool implementing tool.InvokableTool.
type testTool struct{}

func (t *testTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "mock_tool",
		Desc: "A test tool",
	}, nil
}

func (t *testTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	return "tool executed successfully", nil
}

type namedTestTool struct {
	name string
}

func (t *namedTestTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name, Desc: "A named test tool"}, nil
}

func (t *namedTestTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	return "ok", nil
}

// newTestLoop creates a Loop with the given model, maxIterations, and a temp workspace.
func newTestLoop(t *testing.T, chatModel model.ChatModel, maxIterations int) *Loop {
	t.Helper()
	tmpDir := t.TempDir()
	return &Loop{
		bus:           bus.NewMessageBus(1),
		model:         chatModel,
		tools:         tools.NewRegistry(),
		sessions:      session.NewManager(tmpDir),
		context:       NewContextBuilder(tmpDir),
		maxIterations: maxIterations,
		workspacePath: tmpDir,
	}
}

func TestProcessDirect_WithToolCalls(t *testing.T) {
	mockModel := &multiTurnMockModel{}
	loop := newTestLoop(t, mockModel, 10)

	// Register the mock tool so the loop can execute it.
	if err := loop.tools.Register(&testTool{}); err != nil {
		t.Fatalf("failed to register mock tool: %v", err)
	}

	result, err := loop.ProcessDirect(context.Background(), "test message")
	if err != nil {
		t.Fatalf("ProcessDirect returned error: %v", err)
	}

	if result != "Final response" {
		t.Errorf("expected result %q, got %q", "Final response", result)
	}

	if mockModel.callCount != 2 {
		t.Errorf("expected model to be called 2 times, got %d", mockModel.callCount)
	}
}

func TestProcessDirect_LearnsGeoPipelineFromSuccessfulToolSequence(t *testing.T) {
	mockModel := &geoPipelineMockModel{}
	loop := newTestLoop(t, mockModel, 10)

	if err := loop.tools.Register(&namedTestTool{name: "geo_info"}); err != nil {
		t.Fatalf("failed to register geo_info: %v", err)
	}
	if err := loop.tools.Register(&namedTestTool{name: "geo_sinuosity"}); err != nil {
		t.Fatalf("failed to register geo_sinuosity: %v", err)
	}

	result, err := loop.ProcessDirect(context.Background(), "analyze river sinuosity")
	if err != nil {
		t.Fatalf("ProcessDirect returned error: %v", err)
	}
	if result != "Pipeline complete" {
		t.Fatalf("expected final response, got %q", result)
	}

	matches, err := filepath.Glob(filepath.Join(loop.workspacePath, "pipelines", "geo", "*.yaml"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 learned geo pipeline, got %d", len(matches))
	}

	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "geo_info") || !strings.Contains(text, "geo_sinuosity") {
		t.Fatalf("expected learned tool sequence in pipeline record, got %s", text)
	}
}

func TestProcessDirect_MaxIterations(t *testing.T) {
	mockModel := &alwaysToolCallModel{}
	loop := newTestLoop(t, mockModel, 2)

	// Register the mock tool so tool execution does not fail.
	if err := loop.tools.Register(&testTool{}); err != nil {
		t.Fatalf("failed to register mock tool: %v", err)
	}

	result, err := loop.ProcessDirect(context.Background(), "test message")
	if err != nil {
		t.Fatalf("ProcessDirect returned error: %v", err)
	}

	// When maxIterations is exhausted and the model never returns a final text response,
	// the loop falls through with an empty finalContent, which gets replaced by "Processing complete."
	if result != "Processing complete." {
		t.Errorf("expected result %q, got %q", "Processing complete.", result)
	}

	if mockModel.callCount != 2 {
		t.Errorf("expected model to be called exactly 2 times (maxIterations), got %d", mockModel.callCount)
	}
}

func TestRun_IgnoresNilInboundMessage(t *testing.T) {
	loop := newTestLoop(t, nil, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- loop.Run(ctx)
	}()

	loop.bus.PublishInbound(nil)
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected context cancellation error")
		}
		if !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected context canceled error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to return")
	}
}

func TestRun_ExitsWhenInboundClosed(t *testing.T) {
	loop := newTestLoop(t, nil, 1)
	done := make(chan error, 1)

	go func() {
		done <- loop.Run(context.Background())
	}()

	time.Sleep(20 * time.Millisecond)
	loop.bus.Close()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error when inbound channel closes")
		}
		if !strings.Contains(err.Error(), "inbound channel closed") {
			t.Fatalf("expected inbound channel closed error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to return after inbound close")
	}
}

func TestRegisterDefaultTools_WithoutWebSearchKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Web.Search.APIKey = ""

	loop, err := NewLoop(cfg, bus.NewMessageBus(1), nil)
	if err != nil {
		t.Fatalf("NewLoop error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools error: %v", err)
	}

	names := loop.tools.Names()
	if !slices.Contains(names, "web_fetch") {
		t.Fatalf("expected web_fetch to be registered, got: %v", names)
	}
	if !slices.Contains(names, "read_memory") || !slices.Contains(names, "write_memory") || !slices.Contains(names, "append_diary") {
		t.Fatalf("expected memory tools to be registered, got: %v", names)
	}
	if !slices.Contains(names, "web_search") {
		t.Fatalf("expected web_search to be registered (free fallback mode), got: %v", names)
	}
	if !slices.Contains(names, "workflow") {
		t.Fatalf("expected workflow tool to be registered, got: %v", names)
	}
}

func TestRegisterDefaultTools_WithWebSearchKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Web.Search.APIKey = "brave-key"

	loop, err := NewLoop(cfg, bus.NewMessageBus(1), nil)
	if err != nil {
		t.Fatalf("NewLoop error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools error: %v", err)
	}

	names := loop.tools.Names()
	if !slices.Contains(names, "web_search") {
		t.Fatalf("expected web_search to be registered, got: %v", names)
	}
	if !slices.Contains(names, "edit_file") || !slices.Contains(names, "append_file") {
		t.Fatalf("expected edit_file and append_file to be registered, got: %v", names)
	}
}

func TestRegisterDefaultTools_GeoSpatialQuerySkippedWithoutDSN(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Geo.Enabled = true
	cfg.Tools.Geo.PostGISDSN = ""

	loop, err := NewLoop(cfg, bus.NewMessageBus(1), nil)
	if err != nil {
		t.Fatalf("NewLoop error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools error: %v", err)
	}

	names := loop.tools.Names()
	if slices.Contains(names, "geo_spatial_query") {
		t.Fatalf("did not expect geo_spatial_query without DSN, got: %v", names)
	}
	if !slices.Contains(names, "geo_info") || !slices.Contains(names, "geo_process") ||
		!slices.Contains(names, "geo_crs_detect") || !slices.Contains(names, "geo_format_convert") ||
		!slices.Contains(names, "geo_data_catalog") ||
		!slices.Contains(names, "geo_sql_codebook") {
		t.Fatalf("expected baseline geo tools to stay registered, got: %v", names)
	}
}

func TestRegisterDefaultTools_GeoSpatialQueryRegisteredWithDSN(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Geo.Enabled = true
	cfg.Tools.Geo.PostGISDSN = "postgres://golem:golem@localhost:5432/golem?sslmode=disable"

	loop, err := NewLoop(cfg, bus.NewMessageBus(1), nil)
	if err != nil {
		t.Fatalf("NewLoop error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools error: %v", err)
	}

	names := loop.tools.Names()
	if !slices.Contains(names, "geo_spatial_query") {
		t.Fatalf("expected geo_spatial_query to be registered, got: %v", names)
	}
	if !slices.Contains(names, "geo_sql_codebook") {
		t.Fatalf("expected geo_sql_codebook to be registered, got: %v", names)
	}
}

func TestRegisterDefaultTools_LoadsWorkspaceGeoFabricatedTools(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "sinuosity.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("# placeholder\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifestPath := filepath.Join(workspace, "tools", "geo", "geo_sinuosity.yaml")
	manifest := `name: geo_sinuosity
description: Compute sinuosity ratio for river centerlines.
runner: python
script: tools/geo/scripts/sinuosity.py
parameters:
  input_path:
    type: string
    description: Path to the input vector file.
    required: true
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.WorkspaceMode = "path"
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Tools.Geo.Enabled = true

	loop, err := NewLoop(cfg, bus.NewMessageBus(1), nil)
	if err != nil {
		t.Fatalf("NewLoop error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools error: %v", err)
	}

	names := loop.tools.Names()
	if !slices.Contains(names, "geo_sinuosity") {
		t.Fatalf("expected fabricated geo tool to be registered, got: %v", names)
	}
}

func TestProcessForChannel_UsesCustomSessionKey(t *testing.T) {
	loop := newTestLoop(t, nil, 1)
	result, err := loop.ProcessForChannel(context.Background(), "gateway", "s42", "api", "hello")
	if err != nil {
		t.Fatalf("ProcessForChannel error: %v", err)
	}
	if result != "No model configured" {
		t.Fatalf("expected no-model fallback, got %q", result)
	}

	sess := loop.sessions.GetOrCreate("gateway:s42")
	history := sess.GetHistory(0)
	if len(history) != 2 {
		t.Fatalf("expected 2 messages in session, got %d", len(history))
	}
}

func TestRun_HandlesSubagentSystemMessage(t *testing.T) {
	loop := newTestLoop(t, nil, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- loop.Run(ctx)
	}()

	loop.bus.PublishInbound(bus.NewSubagentResultInbound(
		"subagent-1",
		"diag",
		"telegram",
		"chat-1",
		"alice",
		"subagent done",
		"req-1",
		nil,
	))

	select {
	case out := <-loop.bus.Outbound():
		if out.Channel != "telegram" || out.ChatID != "chat-1" {
			t.Fatalf("unexpected outbound route: %+v", out)
		}
		if !strings.Contains(out.Content, "subagent done") {
			t.Fatalf("expected subagent content in outbound message, got: %s", out.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for outbound system relay")
	}

	cancel()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected context canceled from Run, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run shutdown")
	}
}

func TestProcessForChannel_RecordsActivity(t *testing.T) {
	loop := newTestLoop(t, nil, 1)

	var gotChannel, gotChatID string
	loop.SetActivityRecorder(func(channel, chatID string) {
		gotChannel = channel
		gotChatID = chatID
	})

	if _, err := loop.ProcessForChannel(context.Background(), "telegram", "chat-42", "alice", "hello"); err != nil {
		t.Fatalf("ProcessForChannel error: %v", err)
	}

	if gotChannel != "telegram" || gotChatID != "chat-42" {
		t.Fatalf("unexpected activity record: channel=%q chat=%q", gotChannel, gotChatID)
	}
}
