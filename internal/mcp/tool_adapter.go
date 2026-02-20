package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type toolAdapter struct {
	manager    *Manager
	serverName string
	toolName   string
	fullName   string
	desc       string
}

func newToolAdapter(manager *Manager, serverName string, def ToolDefinition) toolAdapter {
	toolName := strings.TrimSpace(def.Name)
	desc := strings.TrimSpace(def.Description)
	if desc == "" {
		desc = toolName
	}

	return toolAdapter{
		manager:    manager,
		serverName: strings.TrimSpace(serverName),
		toolName:   toolName,
		fullName:   fmt.Sprintf("mcp.%s.%s", strings.TrimSpace(serverName), toolName),
		desc:       desc,
	}
}

func (a toolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: a.fullName,
		Desc: a.desc,
		Extra: map[string]any{
			"provider": "mcp",
			"server":   a.serverName,
			"tool":     a.toolName,
		},
	}, nil
}

func (a toolAdapter) InvokableRun(ctx context.Context, argsJSON string, opts ...tool.Option) (string, error) {
	if a.manager == nil {
		return "", fmt.Errorf("mcp manager is not configured")
	}
	return a.manager.CallTool(ctx, a.serverName, a.toolName, argsJSON)
}

func normalizeToolResult(v any) string {
	switch value := v.(type) {
	case nil:
		return "(no output)"
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return "(no output)"
		}
		return text
	case []byte:
		text := strings.TrimSpace(string(value))
		if text == "" {
			return "(no output)"
		}
		return text
	case fmt.Stringer:
		text := strings.TrimSpace(value.String())
		if text == "" {
			return "(no output)"
		}
		return text
	default:
		data, err := json.Marshal(value)
		if err != nil {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text == "" {
				return "(no output)"
			}
			return text
		}
		text := strings.TrimSpace(string(data))
		if text == "" || text == "null" {
			return "(no output)"
		}
		return text
	}
}
