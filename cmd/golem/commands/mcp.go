package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/mcp"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const mcpProbeTimeout = 8 * time.Second // MCP 服务器探测超时时间

var mcpProbeServer = probeMCPServer

// NewMCPCmd 创建 MCP 服务器管理命令。
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
		fmt.Println("No MCP servers configured. Add them in ~/.golem/config.json under 'mcp.servers'.")
		return nil
	}

	var (
		wName   = 20
		wStatus = 12
		wTools  = 8
		wMsg    = 30

		colHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8E4EC6")). // Purple
				Bold(true).
				MarginRight(1)

		nameStyleBase = lipgloss.NewStyle().
				Width(wName).
				MarginRight(1)

		statusStyleBase = lipgloss.NewStyle().
				Width(wStatus).
				MarginRight(1)

		toolsStyleBase = lipgloss.NewStyle().
				Width(wTools).
				MarginRight(1)

		msgStyleBase = lipgloss.NewStyle().
				Width(wMsg).
				MarginRight(1)

		okColor       = lipgloss.Color("#2E8B57") // SeaGreen
		errorColor    = lipgloss.Color("#FF0000") // Red
		disabledColor = lipgloss.Color("241")     // Dark Gray
		defaultColor  = lipgloss.Color("241")
	)

	fmt.Println("MCP servers:")
	fmt.Println()

	headers := lipgloss.JoinHorizontal(lipgloss.Top,
		colHeaderStyle.Width(wName).Render("SERVER"),
		colHeaderStyle.Width(wStatus).Render("STATUS"),
		colHeaderStyle.Width(wTools).Render("TOOLS"),
		colHeaderStyle.Width(wMsg).Render("MESSAGE"),
	)
	fmt.Printf("  %s\n", headers)

	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginRight(1)
	separator := lipgloss.JoinHorizontal(lipgloss.Top,
		sepStyle.Render(strings.Repeat("─", wName)),
		sepStyle.Render(strings.Repeat("─", wStatus)),
		sepStyle.Render(strings.Repeat("─", wTools)),
		sepStyle.Render(strings.Repeat("─", wMsg)),
	)
	fmt.Printf("  %s\n", separator)

	for _, name := range sortedMCPServerNames(cfg.MCP.Servers) {
		serverCfg := cfg.MCP.Servers[name]

		var sStatus string
		var sTools string
		var sMsg string
		sColor := okColor

		if !isConfigMCPServerEnabled(serverCfg) {
			sStatus = "disabled"
			sTools = "-"
			sMsg = ""
			sColor = disabledColor
		} else {
			status, probeErr := probeServerWithTimeout(name, serverCfg)
			if probeErr != nil {
				sStatus = "degraded"
				sTools = "-"
				sMsg = fmt.Sprintf("%v", probeErr)
				sColor = errorColor
			} else if status.Degraded || !status.Connected {
				sStatus = "degraded"
				sTools = "-"
				msg := strings.TrimSpace(status.Message)
				if msg == "" {
					msg = "unknown error"
				}
				sMsg = msg
				sColor = errorColor
			} else {
				sStatus = "connected"
				sTools = fmt.Sprintf("%d", status.ToolCount)
				sMsg = ""
			}
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			nameStyleBase.Render(truncate(name, wName)),
			statusStyleBase.Foreground(sColor).Render(sStatus),
			toolsStyleBase.Foreground(defaultColor).Render(sTools),
			msgStyleBase.Foreground(defaultColor).Render(truncate(sMsg, wMsg)),
		)

		fmt.Printf("  %s\n", row)
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
