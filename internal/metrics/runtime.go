package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const runtimeMetricsFileName = "runtime_metrics.json"

var latencyBucketUpperBoundsMs = []int64{
	10, 25, 50, 100, 250, 500, 1000, 2000, 5000, 10000, 30000,
}

// RuntimeSnapshot contains aggregated runtime metrics for tools and channel sends.
type RuntimeSnapshot struct {
	UpdatedAt time.Time    `json:"updated_at"`
	Tool      ToolStats    `json:"tool"`
	Channel   ChannelStats `json:"channel"`
}

// ToolStats tracks tool execution metrics.
type ToolStats struct {
	Total             int64 `json:"total"`
	Errors            int64 `json:"errors"`
	Timeouts          int64 `json:"timeouts"`
	TotalLatencyMs    int64 `json:"total_latency_ms"`
	MaxLatencyMs      int64 `json:"max_latency_ms"`
	LastLatencyMs     int64 `json:"last_latency_ms"`
	P95ProxyLatencyMs int64 `json:"p95_proxy_latency_ms"`
}

// ErrorRatio returns errors/total in [0,1].
func (t ToolStats) ErrorRatio() float64 {
	if t.Total <= 0 {
		return 0
	}
	return float64(t.Errors) / float64(t.Total)
}

// TimeoutRatio returns timeouts/total in [0,1].
func (t ToolStats) TimeoutRatio() float64 {
	if t.Total <= 0 {
		return 0
	}
	return float64(t.Timeouts) / float64(t.Total)
}

// AvgLatencyMs returns average latency in milliseconds.
func (t ToolStats) AvgLatencyMs() float64 {
	if t.Total <= 0 {
		return 0
	}
	return float64(t.TotalLatencyMs) / float64(t.Total)
}

// ChannelStats tracks outbound channel send metrics.
type ChannelStats struct {
	SendAttempts int64 `json:"send_attempts"`
	SendFailures int64 `json:"send_failures"`
}

// FailureRatio returns failures/attempts in [0,1].
func (c ChannelStats) FailureRatio() float64 {
	if c.SendAttempts <= 0 {
		return 0
	}
	return float64(c.SendFailures) / float64(c.SendAttempts)
}

// HasData reports whether any runtime metrics were recorded.
func (s RuntimeSnapshot) HasData() bool {
	return s.Tool.Total > 0 || s.Channel.SendAttempts > 0
}

// RuntimeMetrics records and persists runtime metrics.
type RuntimeMetrics struct {
	path string

	mu      sync.Mutex
	snap    RuntimeSnapshot
	buckets []int64
}

// NewRuntimeMetrics creates a metrics recorder rooted at <workspace>/state/runtime_metrics.json.
func NewRuntimeMetrics(workspacePath string) *RuntimeMetrics {
	return &RuntimeMetrics{
		path:    runtimeMetricsPath(workspacePath),
		buckets: make([]int64, len(latencyBucketUpperBoundsMs)+1),
	}
}

// Snapshot returns the latest in-memory snapshot.
func (m *RuntimeMetrics) Snapshot() RuntimeSnapshot {
	if m == nil {
		return RuntimeSnapshot{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snap
}

// RecordToolExecution updates tool metrics and persists the snapshot.
func (m *RuntimeMetrics) RecordToolExecution(duration time.Duration, result string, runErr error) (RuntimeSnapshot, error) {
	if m == nil {
		return RuntimeSnapshot{}, nil
	}

	now := time.Now().UTC()
	latencyMs := duration.Milliseconds()
	if latencyMs < 0 {
		latencyMs = 0
	}

	m.mu.Lock()
	m.snap.UpdatedAt = now
	m.snap.Tool.Total++
	m.snap.Tool.TotalLatencyMs += latencyMs
	m.snap.Tool.LastLatencyMs = latencyMs
	if latencyMs > m.snap.Tool.MaxLatencyMs {
		m.snap.Tool.MaxLatencyMs = latencyMs
	}
	if runErr != nil || strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		m.snap.Tool.Errors++
		if isTimeoutError(runErr, result) {
			m.snap.Tool.Timeouts++
		}
	}

	m.buckets[latencyBucketIndex(latencyMs)]++
	m.snap.Tool.P95ProxyLatencyMs = p95ProxyFromBuckets(m.buckets, m.snap.Tool.Total)

	snapshot := m.snap
	m.mu.Unlock()

	return snapshot, persistRuntimeSnapshot(m.path, snapshot)
}

// RecordChannelSend updates outbound channel send metrics and persists the snapshot.
func (m *RuntimeMetrics) RecordChannelSend(success bool) (RuntimeSnapshot, error) {
	if m == nil {
		return RuntimeSnapshot{}, nil
	}

	now := time.Now().UTC()

	m.mu.Lock()
	m.snap.UpdatedAt = now
	m.snap.Channel.SendAttempts++
	if !success {
		m.snap.Channel.SendFailures++
	}
	snapshot := m.snap
	m.mu.Unlock()

	return snapshot, persistRuntimeSnapshot(m.path, snapshot)
}

// ReadRuntimeSnapshot reads the persisted snapshot from workspace state.
// If no file exists yet, it returns a zero-value snapshot and nil error.
func ReadRuntimeSnapshot(workspacePath string) (RuntimeSnapshot, error) {
	path := runtimeMetricsPath(workspacePath)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RuntimeSnapshot{}, nil
		}
		return RuntimeSnapshot{}, fmt.Errorf("read runtime metrics: %w", err)
	}

	var snap RuntimeSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return RuntimeSnapshot{}, fmt.Errorf("decode runtime metrics: %w", err)
	}
	return snap, nil
}

func runtimeMetricsPath(workspacePath string) string {
	return filepath.Join(workspacePath, "state", runtimeMetricsFileName)
}

func persistRuntimeSnapshot(path string, snapshot RuntimeSnapshot) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create runtime metrics dir: %w", err)
	}

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encode runtime metrics: %w", err)
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0o644); err != nil {
		return fmt.Errorf("write runtime metrics temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename runtime metrics file: %w", err)
	}
	return nil
}

func latencyBucketIndex(latencyMs int64) int {
	for i, upper := range latencyBucketUpperBoundsMs {
		if latencyMs <= upper {
			return i
		}
	}
	return len(latencyBucketUpperBoundsMs)
}

func p95ProxyFromBuckets(buckets []int64, total int64) int64 {
	if total <= 0 {
		return 0
	}
	target := int64(float64(total) * 0.95)
	if target <= 0 {
		target = 1
	}

	var cumulative int64
	for i, count := range buckets {
		cumulative += count
		if cumulative < target {
			continue
		}
		if i >= len(latencyBucketUpperBoundsMs) {
			return latencyBucketUpperBoundsMs[len(latencyBucketUpperBoundsMs)-1]
		}
		return latencyBucketUpperBoundsMs[i]
	}
	return latencyBucketUpperBoundsMs[len(latencyBucketUpperBoundsMs)-1]
}

func isTimeoutError(runErr error, result string) bool {
	if errors.Is(runErr, context.DeadlineExceeded) {
		return true
	}
	lowered := strings.ToLower(strings.TrimSpace(fmt.Sprint(runErr)))
	if lowered == "<nil>" {
		lowered = ""
	}
	loweredResult := strings.ToLower(strings.TrimSpace(result))
	combined := lowered + " " + loweredResult
	return strings.Contains(combined, "deadline exceeded") ||
		strings.Contains(combined, "timeout") ||
		strings.Contains(combined, "timed out")
}
