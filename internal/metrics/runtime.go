// Package metrics 实现 Golem 的运行时指标监控，支持工具执行、通道发送及记忆召回的可观察性统计。
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

// latencyBucketUpperBoundsMs 定义了用于 P95 等分位数统计的延迟桶上限（毫秒）。
var latencyBucketUpperBoundsMs = []int64{
	10, 25, 50, 100, 250, 500, 1000, 2000, 5000, 10000, 30000,
}

// RuntimeSnapshot 包含工具、通道及记忆模块的聚合运行时指标快照。
type RuntimeSnapshot struct {
	UpdatedAt time.Time    `json:"updated_at"` // 指标最后更新时间
	Tool      ToolStats    `json:"tool"`       // 工具执行统计
	Channel   ChannelStats `json:"channel"`    // 消息通道发送统计
	Memory    MemoryStats  `json:"memory"`     // 记忆召回统计
}

// ToolStats 跟踪工具执行的各项关键指标。
type ToolStats struct {
	Total             int64 `json:"total"`                // 总调用次数
	Errors            int64 `json:"errors"`               // 失败次数
	Timeouts          int64 `json:"timeouts"`             // 超时次数
	TotalLatencyMs    int64 `json:"total_latency_ms"`     // 累计延迟（毫秒）
	MaxLatencyMs      int64 `json:"max_latency_ms"`       // 最大延迟（毫秒）
	LastLatencyMs     int64 `json:"last_latency_ms"`      // 最近一次执行延迟
	P95ProxyLatencyMs int64 `json:"p95_proxy_latency_ms"` // P95 近似延迟（毫秒）
}

// ErrorRatio 返回工具执行的错误率，范围为 [0, 1]。
func (t ToolStats) ErrorRatio() float64 {
	if t.Total <= 0 {
		return 0
	}
	return float64(t.Errors) / float64(t.Total)
}

// TimeoutRatio 返回工具执行的超时率，范围为 [0, 1]。
func (t ToolStats) TimeoutRatio() float64 {
	if t.Total <= 0 {
		return 0
	}
	return float64(t.Timeouts) / float64(t.Total)
}

// AvgLatencyMs 返回工具执行的平均延迟（毫秒）。
func (t ToolStats) AvgLatencyMs() float64 {
	if t.Total <= 0 {
		return 0
	}
	return float64(t.TotalLatencyMs) / float64(t.Total)
}

// ChannelStats 跟踪出站消息通道的发送指标。
type ChannelStats struct {
	SendAttempts int64 `json:"send_attempts"` // 发送尝试总数
	SendFailures int64 `json:"send_failures"` // 发送失败总数
}

// MemoryStats 跟踪记忆召回系统的可观察性指标。
type MemoryStats struct {
	Recalls          int64 `json:"recalls"`            // 召回请求总数
	TotalItems       int64 `json:"total_items"`        // 累计召回的片段总数
	LastItems        int64 `json:"last_items"`         // 最近一次召回的片段数
	EmptyRecalls     int64 `json:"empty_recalls"`      // 空召回（未找到相关记忆）次数
	LongTermHits     int64 `json:"long_term_hits"`     // 命中长期记忆的次数
	DiaryRecentHits  int64 `json:"diary_recent_hits"`  // 命中最近日记的次数
	DiaryKeywordHits int64 `json:"diary_keyword_hits"` // 通过关键词命中日记的次数
}

// FailureRatio 返回通道发送的失败率，范围为 [0, 1]。
func (c ChannelStats) FailureRatio() float64 {
	if c.SendAttempts <= 0 {
		return 0
	}
	return float64(c.SendFailures) / float64(c.SendAttempts)
}

// HasData 报告快照中是否包含任何已记录的数据。
func (s RuntimeSnapshot) HasData() bool {
	return s.Tool.Total > 0 || s.Channel.SendAttempts > 0
}

// RuntimeMetrics 负责在内存中记录并定期持久化运行时指标。
type RuntimeMetrics struct {
	path string // 持久化文件路径

	mu      sync.Mutex
	snap    RuntimeSnapshot // 内存中的当前快照
	buckets []int64         // 延迟分布桶

	dirty    bool          // 标记是否有未保存的修改
	stopChan chan struct{} // 用于停止刷新协程
	wg       sync.WaitGroup
}

// NewRuntimeMetrics 为指定的工作区创建一个指标记录器，并启动自动持久化协程。
func NewRuntimeMetrics(workspacePath string) *RuntimeMetrics {
	m := &RuntimeMetrics{
		path:     runtimeMetricsPath(workspacePath),
		buckets:  make([]int64, len(latencyBucketUpperBoundsMs)+1),
		stopChan: make(chan struct{}),
	}
	m.wg.Add(1)
	go m.runFlusher()
	return m
}

// runFlusher 是后台运行的刷新协程，定期将修改后的指标同步到磁盘。
func (m *RuntimeMetrics) runFlusher() {
	defer m.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.Flush()
		case <-m.stopChan:
			m.Flush()
			return
		}
	}
}

// Flush 强制将当前内存中的指标快照持久化到磁盘（如果存在修改）。
func (m *RuntimeMetrics) Flush() {
	m.mu.Lock()
	if !m.dirty {
		m.mu.Unlock()
		return
	}
	// 在执行 I/O 操作前释放锁
	snap := m.snap
	m.dirty = false
	m.mu.Unlock()

	_ = persistRuntimeSnapshot(m.path, snap)
}

// Close 优雅地停止刷新器，并确保最后一次指标修改被持久化。
func (m *RuntimeMetrics) Close() error {
	if m == nil {
		return nil
	}
	close(m.stopChan)
	m.wg.Wait()
	return nil
}

// Snapshot 返回当前内存中最新的指标快照副本。
func (m *RuntimeMetrics) Snapshot() RuntimeSnapshot {
	if m == nil {
		return RuntimeSnapshot{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snap
}

// RecordToolExecution 记录一次工具执行的耗时与结果，并更新统计指标。
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
	defer m.mu.Unlock()

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

	m.dirty = true
	return m.snap, nil
}

// RecordChannelSend 记录一次出站通道发送尝试的结果。
func (m *RuntimeMetrics) RecordChannelSend(success bool) (RuntimeSnapshot, error) {
	if m == nil {
		return RuntimeSnapshot{}, nil
	}

	now := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.snap.UpdatedAt = now
	m.snap.Channel.SendAttempts++
	if !success {
		m.snap.Channel.SendFailures++
	}

	m.dirty = true
	return m.snap, nil
}

// RecordMemoryRecall 记录一次记忆召回操作的数量和来源分布。
func (m *RuntimeMetrics) RecordMemoryRecall(itemCount int, sourceHits map[string]int) (RuntimeSnapshot, error) {
	if m == nil {
		return RuntimeSnapshot{}, nil
	}
	if itemCount < 0 {
		itemCount = 0
	}
	now := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.snap.UpdatedAt = now
	m.snap.Memory.Recalls++
	m.snap.Memory.TotalItems += int64(itemCount)
	m.snap.Memory.LastItems = int64(itemCount)
	if itemCount == 0 {
		m.snap.Memory.EmptyRecalls++
	}
	m.snap.Memory.LongTermHits += int64(sourceHits["long_term"])
	m.snap.Memory.DiaryRecentHits += int64(sourceHits["diary_recent"])
	m.snap.Memory.DiaryKeywordHits += int64(sourceHits["diary_keyword"])

	m.dirty = true
	return m.snap, nil
}

// ReadRuntimeSnapshot 从磁盘文件中读取已持久化的运行时指标快照。
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
