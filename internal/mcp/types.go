package mcp

import (
	"context"

	"github.com/MEKXH/golem/internal/config"
)

const (
	TransportStdio   = "stdio"    // 标准输入输出传输协议
	TransportHTTPSSE = "http_sse" // HTTP SSE 传输协议
)

// ToolDefinition 描述从 MCP 服务器发现的工具元数据。
type ToolDefinition struct {
	Name        string // 工具名称
	Description string // 工具功能描述
}

// Client 定义了与 MCP 服务器交互的客户端接口。
type Client interface {
	// ListTools 获取服务器提供的所有工具列表。
	ListTools(ctx context.Context) ([]ToolDefinition, error)
	// CallTool 调用指定的工具并返回结果。
	CallTool(ctx context.Context, toolName, argsJSON string) (any, error)
}

// Connector 定义了建立 MCP 连接并返回客户端实例的接口。
type Connector interface {
	// Connect 根据配置连接到指定的 MCP 服务器。
	Connect(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, error)
}

// Connectors 包含了支持的所有传输协议的连接器实现。
type Connectors struct {
	Stdio   Connector // stdio 传输连接器
	HTTPSSE Connector // http_sse 传输连接器
}

// ServerStatus 表示 MCP 服务器在管理器中的当前运行状态。
type ServerStatus struct {
	Name      string // 服务器名称
	Transport string // 使用的传输协议
	Connected bool   // 是否已成功连接
	Degraded  bool   // 是否处于降级（异常）状态
	ToolCount int    // 发现的工具数量
	Message   string // 状态描述消息或错误信息
}
