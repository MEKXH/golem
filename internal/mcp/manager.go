package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/tools"
)

type serverState struct {
	cfg    config.MCPServerConfig
	client Client
	tools  []ToolDefinition
	status ServerStatus
}

// Manager owns configured MCP servers and dynamic tool registration.
type Manager struct {
	mu         sync.RWMutex
	connectors Connectors
	servers    map[string]*serverState
}

// NewManager constructs a manager from config.MCP.Servers.
func NewManager(servers map[string]config.MCPServerConfig, connectors Connectors) *Manager {
	state := make(map[string]*serverState, len(servers))
	for name, cfg := range servers {
		transport := strings.ToLower(strings.TrimSpace(cfg.Transport))
		cfg.Transport = transport
		state[name] = &serverState{
			cfg: cfg,
			status: ServerStatus{
				Name:      name,
				Transport: transport,
			},
		}
	}

	return &Manager{
		connectors: connectors,
		servers:    state,
	}
}

// DefaultConnectors returns placeholders that degrade servers until real connectors are provided.
func DefaultConnectors() Connectors {
	return Connectors{
		Stdio:   unsupportedConnector{transport: TransportStdio},
		HTTPSSE: unsupportedConnector{transport: TransportHTTPSSE},
	}
}

// Connect discovers tools from each configured server.
// Failures are tracked as degraded states and do not fail the entire manager.
func (m *Manager) Connect(ctx context.Context) error {
	names := m.serverNames()
	for _, name := range names {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		m.mu.RLock()
		state := m.servers[name]
		m.mu.RUnlock()
		if state == nil {
			continue
		}

		connector := m.connectorFor(state.cfg.Transport)
		if connector == nil {
			m.markDegraded(name, fmt.Sprintf("no connector configured for transport %q", state.cfg.Transport))
			continue
		}

		client, err := connector.Connect(ctx, name, state.cfg)
		if err != nil {
			m.markDegraded(name, fmt.Sprintf("connect failed: %v", err))
			continue
		}

		discovered, err := client.ListTools(ctx)
		if err != nil {
			m.markDegraded(name, fmt.Sprintf("list tools failed: %v", err))
			continue
		}

		m.mu.Lock()
		state.client = client
		state.tools = append([]ToolDefinition(nil), discovered...)
		state.status.Connected = true
		state.status.Degraded = false
		state.status.ToolCount = len(discovered)
		state.status.Message = ""
		m.mu.Unlock()
	}
	return nil
}

// RegisterTools registers discovered MCP tools into the given registry.
func (m *Manager) RegisterTools(reg *tools.Registry) error {
	if reg == nil {
		return fmt.Errorf("registry is required")
	}

	toolsToRegister := m.collectRegisteredTools()
	for _, entry := range toolsToRegister {
		if err := reg.Register(entry); err != nil {
			return err
		}
	}
	return nil
}

// CallTool routes a raw tool call to the selected MCP server client.
func (m *Manager) CallTool(ctx context.Context, serverName, toolName, argsJSON string) (string, error) {
	m.mu.RLock()
	state := m.servers[serverName]
	if state == nil {
		m.mu.RUnlock()
		return "", fmt.Errorf("mcp server not found: %s", serverName)
	}
	if state.status.Degraded {
		msg := strings.TrimSpace(state.status.Message)
		if msg == "" {
			msg = "server is degraded"
		}
		m.mu.RUnlock()
		return "", fmt.Errorf("mcp server %s degraded: %s", serverName, msg)
	}
	client := state.client
	m.mu.RUnlock()

	if client == nil {
		return "", fmt.Errorf("mcp server %s is not connected", serverName)
	}

	result, err := client.CallTool(ctx, toolName, argsJSON)
	if err != nil {
		return "", err
	}
	return normalizeToolResult(result), nil
}

// Statuses returns per-server connection/discovery status.
func (m *Manager) Statuses() []ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]ServerStatus, 0, len(names))
	for _, name := range names {
		state := m.servers[name]
		if state == nil {
			continue
		}
		out = append(out, state.status)
	}
	return out
}

func (m *Manager) markDegraded(name, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.servers[name]
	if state == nil {
		return
	}

	state.client = nil
	state.tools = nil
	state.status.Connected = false
	state.status.Degraded = true
	state.status.ToolCount = 0
	state.status.Message = strings.TrimSpace(msg)
}

func (m *Manager) collectRegisteredTools() []toolAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]toolAdapter, 0)
	for _, serverName := range names {
		state := m.servers[serverName]
		if state == nil || state.status.Degraded || state.client == nil {
			continue
		}
		for _, td := range state.tools {
			if strings.TrimSpace(td.Name) == "" {
				continue
			}
			result = append(result, newToolAdapter(m, serverName, td))
		}
	}
	return result
}

func (m *Manager) connectorFor(transport string) Connector {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case TransportStdio:
		return m.connectors.Stdio
	case TransportHTTPSSE:
		return m.connectors.HTTPSSE
	default:
		return nil
	}
}

func (m *Manager) serverNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type unsupportedConnector struct {
	transport string
}

func (c unsupportedConnector) Connect(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, error) {
	return nil, fmt.Errorf("%s connector is not configured", c.transport)
}
