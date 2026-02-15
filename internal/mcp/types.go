package mcp

import (
	"context"

	"github.com/MEKXH/golem/internal/config"
)

const (
	TransportStdio   = "stdio"
	TransportHTTPSSE = "http_sse"
)

// ToolDefinition describes a tool discovered from an MCP server.
type ToolDefinition struct {
	Name        string
	Description string
}

// Client is the MCP client abstraction used by the manager.
type Client interface {
	ListTools(ctx context.Context) ([]ToolDefinition, error)
	CallTool(ctx context.Context, toolName, argsJSON string) (any, error)
}

// Connector dials a server and returns a client implementation.
type Connector interface {
	Connect(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, error)
}

// Connectors groups supported transport connectors.
type Connectors struct {
	Stdio   Connector
	HTTPSSE Connector
}

// ServerStatus represents current manager state for one configured server.
type ServerStatus struct {
	Name      string
	Transport string
	Connected bool
	Degraded  bool
	ToolCount int
	Message   string
}
