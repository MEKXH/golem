package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MEKXH/golem/internal/config"
)

type stdioConnector struct{}

func newStdioConnector() Connector {
	return stdioConnector{}
}

func (c stdioConnector) Connect(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, error) {
	command := strings.TrimSpace(cfg.Command)
	if command == "" {
		return nil, fmt.Errorf("stdio transport requires command")
	}

	cmd := exec.CommandContext(ctx, command, cfg.Args...)
	cmd.Env = mergeEnv(cfg.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start stdio server %q: %w", serverName, err)
	}

	client := &stdioClient{
		serverName: serverName,
		cmd:        cmd,
		stdin:      stdin,
		reader:     bufio.NewReader(stdout),
		stderr:     newTailBuffer(4096),
		exitDone:   make(chan struct{}),
	}

	// Drain stderr to avoid blocking and retain a bounded tail for diagnostics.
	go io.Copy(client.stderr, stderr)
	go func() {
		client.markExited(cmd.Wait())
	}()

	if err := initializeClient(ctx, client); err != nil {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		client.waitForExit(500 * time.Millisecond)
		return nil, client.decorateError(err)
	}
	return client, nil
}

func mergeEnv(extra map[string]string) []string {
	base := os.Environ()
	if len(extra) == 0 {
		return base
	}

	merged := make(map[string]string, len(base)+len(extra))
	for _, item := range base {
		parts := strings.SplitN(item, "=", 2)
		key := parts[0]
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		merged[key] = value
	}
	for key, value := range extra {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		merged[trimmedKey] = value
	}

	out := make([]string, 0, len(merged))
	for key, value := range merged {
		out = append(out, key+"="+value)
	}
	return out
}

type stdioClient struct {
	serverName string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	reader     *bufio.Reader
	stderr     *tailBuffer

	exitMu   sync.RWMutex
	exited   bool
	exitErr  error
	exitDone chan struct{}

	mu     sync.Mutex
	nextID int64
}

func (c *stdioClient) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	result, err := c.invoke(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	return decodeToolDefinitions(result)
}

func (c *stdioClient) CallTool(ctx context.Context, toolName, argsJSON string) (any, error) {
	args, err := parseToolArgs(compactJSONOrRaw(argsJSON))
	if err != nil {
		return nil, err
	}
	result, err := c.invoke(ctx, "tools/call", map[string]any{
		"name":      strings.TrimSpace(toolName),
		"arguments": args,
	})
	if err != nil {
		return nil, err
	}
	return decodeCallResult(result)
}

func (c *stdioClient) invoke(ctx context.Context, method string, params any) (any, error) {
	if err := c.processExitError(); err != nil {
		return nil, c.decorateError(err)
	}

	id := atomic.AddInt64(&c.nextID, 1)
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, fmt.Errorf("encode json-rpc request: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.writeFramed(payload); err != nil {
		return nil, c.decorateError(err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		responsePayload, err := c.readFramed()
		if err != nil {
			return nil, c.decorateError(err)
		}
		result, matched, err := decodeRPCResponse(responsePayload, id)
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		return result, nil
	}
}

func (c *stdioClient) notify(ctx context.Context, method string, params any) error {
	if err := c.processExitError(); err != nil {
		return c.decorateError(err)
	}

	payload, err := json.Marshal(map[string]any{
		"jsonrpc": jsonRPCVersion,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return fmt.Errorf("encode json-rpc notification: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.decorateError(c.writeFramed(payload))
}

func (c *stdioClient) writeFramed(payload []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return fmt.Errorf("write mcp header: %w", err)
	}
	if _, err := c.stdin.Write(payload); err != nil {
		return fmt.Errorf("write mcp payload: %w", err)
	}
	return nil
}

func (c *stdioClient) readFramed() ([]byte, error) {
	contentLength, err := readContentLength(c.reader)
	if err != nil {
		return nil, err
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, body); err != nil {
		return nil, fmt.Errorf("read mcp payload: %w", err)
	}
	return body, nil
}

func (c *stdioClient) markExited(err error) {
	c.exitMu.Lock()
	defer c.exitMu.Unlock()

	if c.exited {
		return
	}
	c.exited = true
	c.exitErr = err
	close(c.exitDone)
}

func (c *stdioClient) waitForExit(timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	select {
	case <-c.exitDone:
	case <-time.After(timeout):
	}
}

func (c *stdioClient) processExitError() error {
	c.exitMu.RLock()
	defer c.exitMu.RUnlock()

	if !c.exited {
		return nil
	}
	if c.exitErr == nil {
		return fmt.Errorf("mcp stdio server %q exited", c.serverName)
	}
	return fmt.Errorf("mcp stdio server %q exited: %w", c.serverName, c.exitErr)
}

func (c *stdioClient) decorateError(err error) error {
	if err == nil {
		return nil
	}

	stderrTail := strings.TrimSpace(c.stderr.String())
	if processErr := c.processExitError(); processErr != nil {
		if stderrTail != "" {
			return fmt.Errorf("%w; process=%v; stderr=%s", err, processErr, stderrTail)
		}
		return fmt.Errorf("%w; process=%v", err, processErr)
	}

	if stderrTail != "" {
		return fmt.Errorf("%w; stderr=%s", err, stderrTail)
	}
	return err
}

type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func newTailBuffer(max int) *tailBuffer {
	if max <= 0 {
		max = 1024
	}
	return &tailBuffer{max: max}
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = append([]byte(nil), b.buf[len(b.buf)-b.max:]...)
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func readContentLength(reader *bufio.Reader) (int, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("read mcp header: %w", err)
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			continue
		}

		value, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid content-length header %q: %w", trimmed, err)
		}
		if value <= 0 {
			return 0, fmt.Errorf("invalid content-length value: %d", value)
		}
		contentLength = value
	}

	if contentLength <= 0 {
		return 0, fmt.Errorf("missing content-length header")
	}
	return contentLength, nil
}
