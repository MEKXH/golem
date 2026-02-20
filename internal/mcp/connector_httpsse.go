package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MEKXH/golem/internal/config"
)

type httpSSEConnector struct {
	client *http.Client
}

const (
	httpSSERequestMaxAttempts = 2
	httpSSERetryBaseBackoff   = 150 * time.Millisecond
)

type retryableError struct {
	err error
}

func (e retryableError) Error() string {
	return e.err.Error()
}

func (e retryableError) Unwrap() error {
	return e.err
}

func makeRetryable(err error) error {
	if err == nil {
		return nil
	}
	return retryableError{err: err}
}

func isRetryable(err error) bool {
	var target retryableError
	return errors.As(err, &target)
}

func newHTTPSSEConnector() Connector {
	return httpSSEConnector{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c httpSSEConnector) Connect(ctx context.Context, serverName string, cfg config.MCPServerConfig) (Client, error) {
	rawURL := strings.TrimSpace(cfg.URL)
	if rawURL == "" {
		return nil, fmt.Errorf("http_sse transport requires url")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid http_sse url %q: %w", rawURL, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported http_sse url scheme: %q", parsedURL.Scheme)
	}

	client := &httpSSEClient{
		httpClient:       c.client,
		sseURL:           parsedURL.String(),
		messageEndpoints: buildCandidateMessageEndpoints(parsedURL),
		headers:          cloneHeaders(cfg.Headers),
	}

	if endpoint, ok := discoverMessageEndpoint(ctx, c.client, parsedURL.String(), client.headers); ok {
		client.messageEndpoints = prependUnique(client.messageEndpoints, endpoint)
	}

	if err := initializeClient(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

func buildCandidateMessageEndpoints(base *url.URL) []string {
	if base == nil {
		return nil
	}

	out := []string{base.String()}
	path := strings.TrimSpace(base.Path)
	if strings.HasSuffix(path, "/sse") {
		alt := *base
		alt.Path = strings.TrimSuffix(path, "/sse") + "/messages"
		out = append(out, alt.String())
	}
	return uniqueStrings(out)
}

func discoverMessageEndpoint(ctx context.Context, client *http.Client, sseURL string, headers map[string]string) (string, bool) {
	discoveryCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		discoveryCtx, cancel = context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(discoveryCtx, http.MethodGet, sseURL, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("Accept", "text/event-stream")
	applyHeaders(req.Header, headers)

	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type"))), "text/event-stream") {
		return "", false
	}

	endpointPath, ok := readSSEEndpointEvent(discoveryCtx, resp.Body)
	if !ok {
		return "", false
	}

	base, err := url.Parse(sseURL)
	if err != nil {
		return "", false
	}
	resolved, err := base.Parse(strings.TrimSpace(endpointPath))
	if err != nil {
		return "", false
	}
	return resolved.String(), true
}

func readSSEEndpointEvent(ctx context.Context, body io.Reader) (string, bool) {
	reader := bufio.NewReader(body)
	eventName := ""
	dataLines := make([]string, 0)

	for {
		select {
		case <-ctx.Done():
			return "", false
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return "", false
		}
		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if len(dataLines) == 0 {
				eventName = ""
				continue
			}
			payload := strings.TrimSpace(strings.Join(dataLines, "\n"))
			if strings.EqualFold(strings.TrimSpace(eventName), "endpoint") && payload != "" {
				return payload, true
			}
			dataLines = dataLines[:0]
			eventName = ""
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
}

type httpSSEClient struct {
	httpClient       *http.Client
	sseURL           string
	messageEndpoints []string
	headers          map[string]string

	mu     sync.Mutex
	nextID int64
}

func (c *httpSSEClient) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	result, err := c.invoke(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	return decodeToolDefinitions(result)
}

func (c *httpSSEClient) CallTool(ctx context.Context, toolName, argsJSON string) (any, error) {
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

func (c *httpSSEClient) invoke(ctx context.Context, method string, params any) (any, error) {
	id := atomic.AddInt64(&c.nextID, 1)

	reqBody, err := json.Marshal(map[string]any{
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

	var lastErr error
	for _, endpoint := range c.messageEndpoints {
		for attempt := 0; attempt < httpSSERequestMaxAttempts; attempt++ {
			result, err := c.postAndReadResponse(ctx, endpoint, reqBody, id)
			if err == nil {
				return result, nil
			}
			lastErr = fmt.Errorf("endpoint=%s attempt=%d/%d: %w", endpoint, attempt+1, httpSSERequestMaxAttempts, err)
			if !isRetryable(err) || attempt == httpSSERequestMaxAttempts-1 {
				break
			}
			if err := waitHTTPSSERetry(ctx, attempt+1); err != nil {
				return nil, err
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no message endpoint available")
	}
	return nil, fmt.Errorf("mcp http_sse invoke %s failed: %w", strings.TrimSpace(method), lastErr)
}

func (c *httpSSEClient) notify(ctx context.Context, method string, params any) error {
	reqBody, err := json.Marshal(map[string]any{
		"jsonrpc": jsonRPCVersion,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return fmt.Errorf("encode json-rpc notification: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for _, endpoint := range c.messageEndpoints {
		for attempt := 0; attempt < httpSSERequestMaxAttempts; attempt++ {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
			if err != nil {
				lastErr = err
				break
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			applyHeaders(req.Header, c.headers)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				lastErr = makeRetryable(err)
				if !isRetryable(lastErr) || attempt == httpSSERequestMaxAttempts-1 {
					break
				}
				if err := waitHTTPSSERetry(ctx, attempt+1); err != nil {
					return err
				}
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}

			statusErr := fmt.Errorf("notification request failed with status %s", resp.Status)
			if shouldRetryHTTPStatus(resp.StatusCode) {
				lastErr = makeRetryable(statusErr)
				if attempt < httpSSERequestMaxAttempts-1 {
					if err := waitHTTPSSERetry(ctx, attempt+1); err != nil {
						return err
					}
					continue
				}
			} else {
				lastErr = statusErr
			}
			break
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no message endpoint available")
	}
	return fmt.Errorf("mcp http_sse notify %s failed: %w", strings.TrimSpace(method), lastErr)
}

func (c *httpSSEClient) postAndReadResponse(ctx context.Context, endpoint string, reqBody []byte, id int64) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	applyHeaders(req.Header, c.headers)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, makeRetryable(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		statusErr := fmt.Errorf("mcp http request failed: %s", msg)
		if shouldRetryHTTPStatus(resp.StatusCode) {
			return nil, makeRetryable(statusErr)
		}
		return nil, statusErr
	}

	contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if strings.HasPrefix(contentType, "text/event-stream") {
		return readRPCResultFromSSE(ctx, resp.Body, id)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read mcp response: %w", err)
	}
	result, matched, err := decodeRPCResponse(payload, id)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("json-rpc response id mismatch")
	}
	return result, nil
}

func shouldRetryHTTPStatus(statusCode int) bool {
	if statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests {
		return true
	}
	return statusCode >= 500 && statusCode <= 599
}

func waitHTTPSSERetry(ctx context.Context, retryIndex int) error {
	if retryIndex <= 0 {
		return nil
	}
	backoff := time.Duration(retryIndex) * httpSSERetryBaseBackoff
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func readRPCResultFromSSE(ctx context.Context, body io.Reader, expectedID int64) (any, error) {
	reader := bufio.NewReader(body)
	dataLines := make([]string, 0)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read sse response: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if len(dataLines) == 0 {
				continue
			}
			payload := strings.TrimSpace(strings.Join(dataLines, "\n"))
			dataLines = dataLines[:0]
			if payload == "" {
				continue
			}
			result, matched, err := decodeRPCResponse([]byte(payload), expectedID)
			if err != nil {
				return nil, err
			}
			if !matched {
				continue
			}
			return result, nil
		}

		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
}

func applyHeaders(dst http.Header, src map[string]string) {
	for key, value := range src {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		dst.Set(trimmedKey, value)
	}
}

func cloneHeaders(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = value
	}
	return out
}

func prependUnique(items []string, first string) []string {
	result := make([]string, 0, len(items)+1)
	trimmed := strings.TrimSpace(first)
	if trimmed != "" {
		result = append(result, trimmed)
	}
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if value == trimmed {
			continue
		}
		result = append(result, value)
	}
	return uniqueStrings(result)
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
