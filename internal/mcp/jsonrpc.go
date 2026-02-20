package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const jsonRPCVersion = "2.0"

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func decodeToolDefinitions(result any) ([]ToolDefinition, error) {
	if result == nil {
		return nil, nil
	}

	var toolsValue any
	switch value := result.(type) {
	case map[string]any:
		toolsValue = value["tools"]
	default:
		toolsValue = value
	}

	items, ok := toolsValue.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected tools/list result shape")
	}

	defs := make([]ToolDefinition, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(stringValue(obj["name"]))
		if name == "" {
			continue
		}
		defs = append(defs, ToolDefinition{
			Name:        name,
			Description: strings.TrimSpace(stringValue(obj["description"])),
		})
	}
	return defs, nil
}

func parseToolArgs(argsJSON string) (any, error) {
	trimmed := strings.TrimSpace(argsJSON)
	if trimmed == "" {
		return map[string]any{}, nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, fmt.Errorf("invalid tool args json: %w", err)
	}
	if parsed == nil {
		return map[string]any{}, nil
	}
	return parsed, nil
}

func decodeCallResult(result any) (any, error) {
	obj, ok := result.(map[string]any)
	if !ok {
		return result, nil
	}

	isErr, _ := obj["isError"].(bool)
	if text := extractTextContent(obj["content"]); text != "" {
		if isErr {
			return nil, errors.New(text)
		}
		return text, nil
	}
	if isErr {
		return nil, fmt.Errorf("mcp tool call failed")
	}

	if structured, ok := obj["structuredContent"]; ok && structured != nil {
		return structured, nil
	}
	return result, nil
}

func extractTextContent(v any) string {
	items, ok := v.([]any)
	if !ok {
		return ""
	}

	parts := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.ToLower(strings.TrimSpace(stringValue(obj["type"]))) != "text" {
			continue
		}
		text := strings.TrimSpace(stringValue(obj["text"]))
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	switch value := v.(type) {
	case string:
		return value
	default:
		return fmt.Sprint(v)
	}
}

func decodeRPCResponse(payload []byte, expectedID int64) (any, bool, error) {
	var envelope map[string]any
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, false, fmt.Errorf("decode json-rpc response: %w", err)
	}

	// Notifications/messages without ID can be ignored while waiting for the response.
	if _, hasID := envelope["id"]; !hasID {
		return nil, false, nil
	}

	if normalizeRPCID(envelope["id"]) != normalizeRPCID(expectedID) {
		return nil, false, nil
	}

	if errValue, ok := envelope["error"]; ok && errValue != nil {
		parsedErr := rpcError{}
		if raw, err := json.Marshal(errValue); err == nil {
			_ = json.Unmarshal(raw, &parsedErr)
		}
		msg := strings.TrimSpace(parsedErr.Message)
		if msg == "" {
			msg = strings.TrimSpace(fmt.Sprint(errValue))
		}
		if msg == "" {
			msg = "json-rpc request failed"
		}
		return nil, true, errors.New(msg)
	}

	return envelope["result"], true, nil
}

func normalizeRPCID(id any) string {
	switch value := id.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case float64:
		return fmt.Sprintf("%.0f", value)
	case int:
		return fmt.Sprintf("%d", value)
	case int64:
		return fmt.Sprintf("%d", value)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func buildInitializeParams() map[string]any {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "golem",
			"version": "v0.4.0",
		},
	}
}

func compactJSONOrRaw(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}
	var out bytes.Buffer
	if err := json.Compact(&out, []byte(trimmed)); err != nil {
		return trimmed
	}
	return out.String()
}

type rpcInvoker interface {
	invoke(ctx context.Context, method string, params any) (any, error)
	notify(ctx context.Context, method string, params any) error
}

func initializeClient(ctx context.Context, invoker rpcInvoker) error {
	if _, err := invoker.invoke(ctx, "initialize", buildInitializeParams()); err != nil {
		return fmt.Errorf("initialize mcp session: %w", err)
	}
	if err := invoker.notify(ctx, "notifications/initialized", map[string]any{}); err != nil {
		return fmt.Errorf("send initialized notification: %w", err)
	}
	return nil
}
