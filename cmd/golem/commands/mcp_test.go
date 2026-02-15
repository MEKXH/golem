package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/mcp"
)

func TestMCPDisable_SetsServerDisabled(t *testing.T) {
	prepareMCPWorkspace(t)
	seedMCPServerConfig(t)

	if err := runMCPDisable(nil, []string{"localfs"}); err != nil {
		t.Fatalf("runMCPDisable: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	srv := cfg.MCP.Servers["localfs"]
	if isConfigMCPServerEnabled(srv) {
		t.Fatalf("expected localfs server disabled, got %+v", srv)
	}
}

func TestMCPStatus_ShowsDisabledServer(t *testing.T) {
	prepareMCPWorkspace(t)
	seedMCPServerConfig(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	srv := cfg.MCP.Servers["localfs"]
	disabled := false
	srv.Enabled = &disabled
	cfg.MCP.Servers["localfs"] = srv
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	output := captureOutput(t, func() {
		if err := runMCPStatus(nil, nil); err != nil {
			t.Fatalf("runMCPStatus: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(output), "disabled") {
		t.Fatalf("expected disabled status in output, got: %s", output)
	}
}

func TestMCPReconnect_UsesProbeAndReportsHealthy(t *testing.T) {
	prepareMCPWorkspace(t)
	seedMCPServerConfig(t)

	origProbe := mcpProbeServer
	mcpProbeServer = func(ctx context.Context, serverName string, cfg config.MCPServerConfig) (mcp.ServerStatus, error) {
		return mcp.ServerStatus{
			Name:      serverName,
			Transport: cfg.Transport,
			Connected: true,
			Degraded:  false,
			ToolCount: 2,
		}, nil
	}
	defer func() { mcpProbeServer = origProbe }()

	output := captureOutput(t, func() {
		if err := runMCPReconnect(nil, []string{"localfs"}); err != nil {
			t.Fatalf("runMCPReconnect: %v", err)
		}
	})

	if !strings.Contains(output, "reconnected") {
		t.Fatalf("expected reconnect success output, got: %s", output)
	}
}

func TestMCPCommand_RegisteredInRoot(t *testing.T) {
	root := NewRootCmd()
	found, _, err := root.Find([]string{"mcp", "status"})
	if err != nil {
		t.Fatalf("find mcp status command: %v", err)
	}
	if found == nil || found.Name() != "status" {
		t.Fatalf("expected status command, got %#v", found)
	}
}

func prepareMCPWorkspace(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}
}

func seedMCPServerConfig(t *testing.T) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	cfg.MCP.Servers = map[string]config.MCPServerConfig{
		"localfs": {
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
		},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save: %v", err)
	}
}
