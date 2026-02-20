package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/mcp"
	"github.com/spf13/cobra"
)

const mcpProbeTimeout = 8 * time.Second

var mcpProbeServer = probeMCPServer

func NewMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP servers",
	}

	cmd.AddCommand(
		newMCPStatusCmd(),
		newMCPReconnectCmd(),
		newMCPDisableCmd(),
	)

	return cmd
}

func newMCPStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show MCP server health and degraded reasons",
		RunE:  runMCPStatus,
	}
}

func newMCPReconnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reconnect <server>",
		Short: "Probe and reconnect one MCP server",
		Args:  cobra.ExactArgs(1),
		RunE:  runMCPReconnect,
	}
}

func newMCPDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <server>",
		Short: "Disable an MCP server in config",
		Args:  cobra.ExactArgs(1),
		RunE:  runMCPDisable,
	}
}

func runMCPStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.MCP.Servers) == 0 {
		fmt.Println("No MCP servers configured.")
		return nil
	}

	fmt.Println("MCP servers:")
	for _, name := range sortedMCPServerNames(cfg.MCP.Servers) {
		serverCfg := cfg.MCP.Servers[name]
		if !isConfigMCPServerEnabled(serverCfg) {
			fmt.Printf("  %s: disabled\n", name)
			continue
		}

		status, probeErr := probeServerWithTimeout(name, serverCfg)
		if probeErr != nil {
			fmt.Printf("  %s: degraded (%v)\n", name, probeErr)
			continue
		}

		if status.Degraded || !status.Connected {
			msg := strings.TrimSpace(status.Message)
			if msg == "" {
				msg = "unknown error"
			}
			fmt.Printf("  %s: degraded (%s)\n", name, msg)
			continue
		}

		fmt.Printf("  %s: connected (tools=%d)\n", name, status.ToolCount)
	}

	return nil
}

func runMCPReconnect(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	serverCfg, ok := cfg.MCP.Servers[serverName]
	if !ok {
		return fmt.Errorf("mcp server not found: %s", serverName)
	}
	if !isConfigMCPServerEnabled(serverCfg) {
		return fmt.Errorf("mcp server %s is disabled in config", serverName)
	}

	status, probeErr := probeServerWithTimeout(serverName, serverCfg)
	if probeErr != nil {
		return fmt.Errorf("reconnect %s failed: %w", serverName, probeErr)
	}
	if status.Degraded || !status.Connected {
		msg := strings.TrimSpace(status.Message)
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("mcp server %s is still degraded: %s", serverName, msg)
	}

	fmt.Printf("MCP server %s reconnected (tools=%d).\n", serverName, status.ToolCount)
	return nil
}

func runMCPDisable(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	serverCfg, ok := cfg.MCP.Servers[serverName]
	if !ok {
		return fmt.Errorf("mcp server not found: %s", serverName)
	}

	disabled := false
	serverCfg.Enabled = &disabled
	cfg.MCP.Servers[serverName] = serverCfg
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("MCP server %s disabled in config.\n", serverName)
	return nil
}

func probeServerWithTimeout(serverName string, cfg config.MCPServerConfig) (mcp.ServerStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mcpProbeTimeout)
	defer cancel()

	return mcpProbeServer(ctx, serverName, cfg)
}

func probeMCPServer(ctx context.Context, serverName string, cfg config.MCPServerConfig) (mcp.ServerStatus, error) {
	mgr := mcp.NewManager(
		map[string]config.MCPServerConfig{serverName: cfg},
		mcp.DefaultConnectors(),
	)
	if err := mgr.Connect(ctx); err != nil {
		return mcp.ServerStatus{}, err
	}

	statuses := mgr.Statuses()
	if len(statuses) == 0 {
		return mcp.ServerStatus{
			Name:      serverName,
			Transport: strings.ToLower(strings.TrimSpace(cfg.Transport)),
			Connected: false,
			Degraded:  true,
			Message:   "no status available",
		}, nil
	}
	return statuses[0], nil
}

func sortedMCPServerNames(servers map[string]config.MCPServerConfig) []string {
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func isConfigMCPServerEnabled(server config.MCPServerConfig) bool {
	return config.IsMCPServerEnabled(server)
}
