package mcp

import (
	"context"

	"github.com/MEKXH/golem/internal/config"
)

const (
	TransportStdio   = "stdio"
	TransportHTTPSSE = "http_sse"
)

// ToolDefinition 描述从 MCP 服务器发现的工具。
type ToolDefinition struct {
	Name        string
	Description string
}

// Client 是管理器使用的 MCP 客户端抽象。
type Client interface {
	ListTools(ctx context.Context) ([]ToolDefinition, error)
	CallTool(ctx context.Context, toolName, argsJSON string) (any, error)
}

// Connector 拨打服务器并返回客户端实现。
type Connector interface {
	Connect(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, error)
}

// Connectors 组合支持的传输连接器。
type Connectors struct {
	Stdio   Connector
	HTTPSSE Connector
}

// ServerStatus 表示一个已配置服务器的当前管理器状态。
type ServerStatus struct {
	Name      string
	Transport string
	Connected bool
	Degraded  bool
	ToolCount int
	Message   string
}
