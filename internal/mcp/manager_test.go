package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/tools"
)

type fakeConnector struct {
	client Client
	err    error
	calls  int
}

func (f *fakeConnector) Connect(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.client, nil
}

type fakeCall struct {
	toolName string
	argsJSON string
}

type fakeClient struct {
	tools      []ToolDefinition
	listErr    error
	callErr    error
	callResult any
	calls      []fakeCall
}

func (f *fakeClient) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.tools, nil
}

type sequenceConnector struct {
	results []fakeConnectorResult
	calls   int
}

type fakeConnectorResult struct {
	client Client
	err    error
}

func (s *sequenceConnector) Connect(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, error) {
	index := s.calls
	s.calls++
	if index >= len(s.results) {
		last := s.results[len(s.results)-1]
		return last.client, last.err
	}
	return s.results[index].client, s.results[index].err
}

func (f *fakeClient) CallTool(ctx context.Context, toolName string, argsJSON string) (any, error) {
	f.calls = append(f.calls, fakeCall{
		toolName: toolName,
		argsJSON: argsJSON,
	})
	if f.callErr != nil {
		return nil, f.callErr
	}
	return f.callResult, nil
}

func TestManager_RegisterTools_FromAvailableServer(t *testing.T) {
	client := &fakeClient{
		tools: []ToolDefinition{
			{Name: "read", Description: "Read from MCP server"},
		},
		callResult: "ok",
	}

	mgr := NewManager(
		map[string]config.MCPServerConfig{
			"localfs": {
				Transport: "stdio",
				Command:   "localfs-mcp",
			},
		},
		Connectors{
			Stdio:   &fakeConnector{client: client},
			HTTPSSE: &fakeConnector{err: errors.New("unexpected transport")},
		},
	)

	if err := mgr.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	reg := tools.NewRegistry()
	if err := mgr.RegisterTools(reg); err != nil {
		t.Fatalf("RegisterTools() error: %v", err)
	}

	toolName := "mcp.localfs.read"
	if _, ok := reg.Get(toolName); !ok {
		t.Fatalf("expected tool %q to be registered", toolName)
	}

	argsJSON := `{"path":"notes/todo.md"}`
	result, err := reg.Execute(context.Background(), toolName, argsJSON)
	if err != nil {
		t.Fatalf("registry execute error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected MCP result %q, got %q", "ok", result)
	}

	if len(client.calls) != 1 {
		t.Fatalf("expected one MCP call, got %d", len(client.calls))
	}
	if client.calls[0].toolName != "read" {
		t.Fatalf("expected MCP tool name %q, got %q", "read", client.calls[0].toolName)
	}
	if client.calls[0].argsJSON != argsJSON {
		t.Fatalf("expected raw args JSON %q, got %q", argsJSON, client.calls[0].argsJSON)
	}
}

func TestManager_DegradedState_WhenConnectorFails(t *testing.T) {
	badErr := errors.New("dial tcp: connection refused")

	goodClient := &fakeClient{
		tools: []ToolDefinition{
			{Name: "ping", Description: "Ping tool"},
		},
		callResult: "pong",
	}

	mgr := NewManager(
		map[string]config.MCPServerConfig{
			"broken": {
				Transport: "http_sse",
				URL:       "http://127.0.0.1:9011/sse",
			},
			"ok": {
				Transport: "stdio",
				Command:   "ok-mcp",
			},
		},
		Connectors{
			Stdio:   &fakeConnector{client: goodClient},
			HTTPSSE: &fakeConnector{err: badErr},
		},
	)

	if err := mgr.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() should not fail entire manager, got: %v", err)
	}

	statuses := mgr.Statuses()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 server statuses, got %d", len(statuses))
	}

	var brokenStatus ServerStatus
	foundBroken := false
	for _, st := range statuses {
		if st.Name == "broken" {
			brokenStatus = st
			foundBroken = true
			break
		}
	}
	if !foundBroken {
		t.Fatal("expected status entry for broken server")
	}
	if !brokenStatus.Degraded {
		t.Fatal("expected broken server to be degraded")
	}
	if !strings.Contains(brokenStatus.Message, badErr.Error()) {
		t.Fatalf("expected degraded message to include %q, got %q", badErr.Error(), brokenStatus.Message)
	}

	reg := tools.NewRegistry()
	if err := mgr.RegisterTools(reg); err != nil {
		t.Fatalf("RegisterTools() error: %v", err)
	}

	if _, ok := reg.Get("mcp.broken.ping"); ok {
		t.Fatal("did not expect tools from degraded server to be registered")
	}
	if _, ok := reg.Get("mcp.ok.ping"); !ok {
		t.Fatal("expected tools from healthy server to be registered")
	}
}

func TestManager_CallTool_ReconnectsAfterCallFailure(t *testing.T) {
	brokenClient := &fakeClient{
		tools: []ToolDefinition{
			{Name: "echo", Description: "Echo"},
		},
		callErr: errors.New("connection reset by peer"),
	}
	recoveredClient := &fakeClient{
		tools: []ToolDefinition{
			{Name: "echo", Description: "Echo"},
		},
		callResult: "ok-after-reconnect",
	}
	connector := &sequenceConnector{
		results: []fakeConnectorResult{
			{client: brokenClient},
			{client: recoveredClient},
		},
	}

	mgr := NewManager(
		map[string]config.MCPServerConfig{
			"remote": {
				Transport: "http_sse",
				URL:       "http://127.0.0.1:19001/sse",
			},
		},
		Connectors{
			Stdio:   &fakeConnector{},
			HTTPSSE: connector,
		},
	)

	if err := mgr.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	result, err := mgr.CallTool(context.Background(), "remote", "echo", `{}`)
	if err != nil {
		t.Fatalf("CallTool() error: %v", err)
	}
	if result != "ok-after-reconnect" {
		t.Fatalf("expected reconnect result, got %q", result)
	}
	if connector.calls < 2 {
		t.Fatalf("expected reconnect to trigger a second connect attempt, got %d calls", connector.calls)
	}

	status := mgr.Statuses()[0]
	if status.Degraded {
		t.Fatalf("expected recovered status, got degraded: %+v", status)
	}
}

func TestManager_CallTool_RecoversFromStartupDegradedState(t *testing.T) {
	recoveredClient := &fakeClient{
		tools: []ToolDefinition{
			{Name: "echo", Description: "Echo"},
		},
		callResult: "pong",
	}
	connector := &sequenceConnector{
		results: []fakeConnectorResult{
			{err: errors.New("connect timeout")},
			{client: recoveredClient},
		},
	}

	mgr := NewManager(
		map[string]config.MCPServerConfig{
			"remote": {
				Transport: "http_sse",
				URL:       "http://127.0.0.1:19002/sse",
			},
		},
		Connectors{
			Stdio:   &fakeConnector{},
			HTTPSSE: connector,
		},
	)

	if err := mgr.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() should not fail entire manager: %v", err)
	}

	before := mgr.Statuses()[0]
	if !before.Degraded {
		t.Fatalf("expected degraded status before recovery, got %+v", before)
	}

	result, err := mgr.CallTool(context.Background(), "remote", "echo", `{}`)
	if err != nil {
		t.Fatalf("CallTool() should recover degraded server, got error: %v", err)
	}
	if result != "pong" {
		t.Fatalf("expected pong result after recovery, got %q", result)
	}

	after := mgr.Statuses()[0]
	if after.Degraded || !after.Connected {
		t.Fatalf("expected connected healthy status after recovery, got %+v", after)
	}
}

func TestManager_NewManager_SkipsDisabledServers(t *testing.T) {
	disabled := false
	mgr := NewManager(
		map[string]config.MCPServerConfig{
			"disabled": {
				Enabled:   &disabled,
				Transport: "stdio",
				Command:   "disabled-mcp",
			},
			"enabled": {
				Transport: "stdio",
				Command:   "enabled-mcp",
			},
		},
		Connectors{
			Stdio:   &fakeConnector{},
			HTTPSSE: &fakeConnector{},
		},
	)

	statuses := mgr.Statuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 server status, got %d", len(statuses))
	}
	if statuses[0].Name != "enabled" {
		t.Fatalf("expected enabled server to remain, got %+v", statuses[0])
	}
}
