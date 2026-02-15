package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/config"
)

func TestStdioConnector_ConnectAndCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connector := newStdioConnector()
	client, err := connector.Connect(ctx, "helper", config.MCPServerConfig{
		Transport: "stdio",
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPHelperProcess", "--", "mcp-stdio-helper"},
		Env: map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
		},
	})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("unexpected tool definitions: %+v", tools)
	}

	result, err := client.CallTool(context.Background(), "echo", `{"message":"hello"}`)
	if err != nil {
		t.Fatalf("CallTool() error: %v", err)
	}
	if got := strings.TrimSpace(fmt.Sprint(result)); got != "echo: hello" {
		t.Fatalf("unexpected tool result: %v", result)
	}
}

func TestHTTPSSEConnector_ConnectDiscoverAndCall(t *testing.T) {
	var receivedHeader string

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Test-Token")
		w.Header().Set("Content-Type", "text/event-stream")
		if flusher, ok := w.(http.Flusher); ok {
			_, _ = io.WriteString(w, "event: endpoint\ndata: /rpc\n\n")
			flusher.Flush()
			return
		}
		_, _ = io.WriteString(w, "event: endpoint\ndata: /rpc\n\n")
	})
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test-Token"); got == "" {
			t.Errorf("expected custom header on RPC request")
		}

		defer r.Body.Close()
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		method := strings.TrimSpace(stringValue(req["method"]))
		id, hasID := req["id"]
		if !hasID {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		var result any
		switch method {
		case "initialize":
			result = map[string]any{
				"capabilities": map[string]any{},
				"serverInfo": map[string]any{
					"name":    "test-http-sse",
					"version": "1.0.0",
				},
			}
		case "tools/list":
			result = map[string]any{
				"tools": []map[string]any{
					{
						"name":        "echo",
						"description": "Echo tool",
					},
				},
			}
		case "tools/call":
			result = map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": "echo: from-http",
					},
				},
			}
		default:
			result = map[string]any{}
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	connector := newHTTPSSEConnector()
	client, err := connector.Connect(context.Background(), "remote", config.MCPServerConfig{
		Transport: "http_sse",
		URL:       server.URL + "/sse",
		Headers: map[string]string{
			"X-Test-Token": "abc123",
		},
	})
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	if receivedHeader != "abc123" {
		t.Fatalf("expected header on SSE discovery request, got %q", receivedHeader)
	}

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("unexpected tool definitions: %+v", tools)
	}

	result, err := client.CallTool(context.Background(), "echo", `{}`)
	if err != nil {
		t.Fatalf("CallTool() error: %v", err)
	}
	if got := strings.TrimSpace(fmt.Sprint(result)); got != "echo: from-http" {
		t.Fatalf("unexpected tool result: %v", result)
	}
}

func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	isHelper := false
	for _, arg := range os.Args {
		if arg == "mcp-stdio-helper" {
			isHelper = true
			break
		}
	}
	if !isHelper {
		return
	}

	runMCPHelperProcess()
	os.Exit(0)
}

func runMCPHelperProcess() {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	for {
		contentLength, err := readContentLength(reader)
		if err != nil {
			return
		}
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, body); err != nil {
			return
		}

		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			return
		}

		method := strings.TrimSpace(stringValue(req["method"]))
		id, hasID := req["id"]
		if !hasID {
			continue
		}

		var result any
		switch method {
		case "initialize":
			result = map[string]any{
				"capabilities": map[string]any{},
				"serverInfo": map[string]any{
					"name":    "test-stdio",
					"version": "1.0.0",
				},
			}
		case "tools/list":
			result = map[string]any{
				"tools": []map[string]any{
					{
						"name":        "echo",
						"description": "Echo tool",
					},
				},
			}
		case "tools/call":
			text := "echo: "
			if params, ok := req["params"].(map[string]any); ok {
				if args, ok := params["arguments"].(map[string]any); ok {
					text += strings.TrimSpace(stringValue(args["message"]))
				}
			}
			result = map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": text,
					},
				},
			}
		default:
			result = map[string]any{}
		}

		resp, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		})
		_, _ = io.WriteString(writer, fmt.Sprintf("Content-Length: %d\r\n\r\n", len(resp)))
		_, _ = writer.Write(resp)
	}
}

func TestReadSSEEndpointEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	payload := "event: endpoint\ndata: /rpc\n\n"
	endpoint, ok := readSSEEndpointEvent(ctx, strings.NewReader(payload))
	if !ok {
		t.Fatal("expected endpoint event")
	}
	if endpoint != "/rpc" {
		t.Fatalf("expected /rpc, got %q", endpoint)
	}
}
