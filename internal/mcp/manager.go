package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/tools"
)

const (
	reconnectMaxAttempts = 3
	reconnectBaseBackoff = 250 * time.Millisecond
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
		if !config.IsMCPServerEnabled(cfg) {
			continue
		}
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

// DefaultConnectors returns production connectors for stdio and HTTP/SSE transports.
func DefaultConnectors() Connectors {
	return Connectors{
		Stdio:   newStdioConnector(),
		HTTPSSE: newHTTPSSEConnector(),
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

		cfg, ok := m.serverConfig(name)
		if !ok {
			continue
		}

		client, discovered, err := m.connectAndDiscover(ctx, name, cfg)
		if err != nil {
			m.markDegraded(name, fmt.Sprintf("connect failed: %v", err))
			continue
		}
		m.markConnected(name, client, discovered, "")
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
// When the server is degraded or a tool call fails, CallTool attempts bounded reconnect with backoff.
func (m *Manager) CallTool(ctx context.Context, serverName, toolName, argsJSON string) (string, error) {
	client, err := m.ensureConnectedClient(ctx, serverName)
	if err != nil {
		return "", err
	}

	result, callErr := client.CallTool(ctx, toolName, argsJSON)
	if callErr == nil {
		return normalizeToolResult(result), nil
	}

	reconnectErr := m.reconnectServer(ctx, serverName, fmt.Sprintf("tool call failed: %v", callErr))
	if reconnectErr != nil {
		return "", fmt.Errorf("mcp server %s call failed: %v; reconnect failed: %w", serverName, callErr, reconnectErr)
	}

	client, err = m.currentClient(serverName)
	if err != nil {
		return "", err
	}
	result, callErr = client.CallTool(ctx, toolName, argsJSON)
	if callErr != nil {
		m.markDegraded(serverName, fmt.Sprintf("tool call failed after reconnect: %v", callErr))
		return "", fmt.Errorf("mcp server %s call failed after reconnect: %w", serverName, callErr)
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

func (m *Manager) ensureConnectedClient(ctx context.Context, serverName string) (Client, error) {
	m.mu.RLock()
	state := m.servers[serverName]
	if state == nil {
		m.mu.RUnlock()
		return nil, fmt.Errorf("mcp server not found: %s", serverName)
	}

	shouldReconnect := state.status.Degraded || state.client == nil
	reason := strings.TrimSpace(state.status.Message)
	client := state.client
	m.mu.RUnlock()

	if !shouldReconnect {
		return client, nil
	}
	if reason == "" {
		reason = "server not connected"
	}

	if err := m.reconnectServer(ctx, serverName, reason); err != nil {
		return nil, fmt.Errorf("mcp server %s unavailable: %w", serverName, err)
	}
	return m.currentClient(serverName)
}

func (m *Manager) currentClient(serverName string) (Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := m.servers[serverName]
	if state == nil {
		return nil, fmt.Errorf("mcp server not found: %s", serverName)
	}
	if state.client == nil {
		return nil, fmt.Errorf("mcp server %s is not connected", serverName)
	}
	return state.client, nil
}

func (m *Manager) reconnectServer(ctx context.Context, serverName, reason string) error {
	cfg, ok := m.serverConfig(serverName)
	if !ok {
		return fmt.Errorf("mcp server not found: %s", serverName)
	}

	var lastErr error
	for attempt := 1; attempt <= reconnectMaxAttempts; attempt++ {
		if attempt > 1 {
			if err := waitReconnectBackoff(ctx, attempt-1); err != nil {
				return err
			}
		}

		client, discovered, err := m.connectAndDiscover(ctx, serverName, cfg)
		if err == nil {
			recoveredMsg := fmt.Sprintf("recovered after %d reconnect attempt(s)", attempt)
			m.markConnected(serverName, client, discovered, recoveredMsg)
			return nil
		}
		lastErr = err
	}

	degradedMsg := fmt.Sprintf("%s; reconnect failed after %d attempts: %v", strings.TrimSpace(reason), reconnectMaxAttempts, lastErr)
	m.markDegraded(serverName, degradedMsg)
	return fmt.Errorf("reconnect failed after %d attempts: %w", reconnectMaxAttempts, lastErr)
}

func waitReconnectBackoff(ctx context.Context, retryIndex int) error {
	if retryIndex <= 0 {
		return nil
	}

	delay := time.Duration(retryIndex) * reconnectBaseBackoff
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (m *Manager) connectAndDiscover(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, []ToolDefinition, error) {
	connector := m.connectorFor(cfg.Transport)
	if connector == nil {
		return nil, nil, fmt.Errorf("no connector configured for transport %q", cfg.Transport)
	}

	client, err := connector.Connect(ctx, serverName, cfg)
	if err != nil {
		return nil, nil, err
	}

	discovered, err := client.ListTools(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list tools failed: %w", err)
	}
	return client, discovered, nil
}

func (m *Manager) markConnected(name string, client Client, discovered []ToolDefinition, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.servers[name]
	if state == nil {
		return
	}

	state.client = client
	state.tools = append([]ToolDefinition(nil), discovered...)
	state.status.Connected = true
	state.status.Degraded = false
	state.status.ToolCount = len(discovered)
	state.status.Message = strings.TrimSpace(message)
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

func (m *Manager) serverConfig(name string) (config.MCPServerConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := m.servers[name]
	if state == nil {
		return config.MCPServerConfig{}, false
	}
	return state.cfg, true
}
