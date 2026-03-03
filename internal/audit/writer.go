// Package audit 实现 Golem 的运行时审计日志记录功能。
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	auditFileMode = 0644 // 审计文件的默认权限
	auditDirMode  = 0755 // 审计目录的默认权限
)

// Event 表示单条审计记录，以 JSON 行 (JSONL) 格式写入。
type Event struct {
	Time      time.Time `json:"time"`                 // 事件发生时间
	Type      string    `json:"type"`                 // 事件类型（如 policy_allow, tool_execution）
	RequestID string    `json:"request_id,omitempty"` // 请求追踪 ID
	Tool      string    `json:"tool,omitempty"`       // 关联的工具名称
	Result    string    `json:"result,omitempty"`     // 执行结果或决策状态
}

// Writer 负责将审计事件异步追加到工作区内的 <workspace>/state/audit.jsonl 文件中。
type Writer struct {
	path string     // 审计文件路径
	mu   sync.Mutex // 确保并发写入的线程安全
}

// NewWriter 在工作区的 state 目录下创建一个新的审计日志写入器。
func NewWriter(workspace string) *Writer {
	return &Writer{
		path: filepath.Join(workspace, "state", "audit.jsonl"),
	}
}

// Append 将一个审计事件作为一行 JSON 数据追加到文件中。
func (w *Writer) Append(event Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(w.path), auditDirMode); err != nil {
		return fmt.Errorf("create audit dir: %w", err)
	}

	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, auditFileMode)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer file.Close()

	encoded, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}
	encoded = append(encoded, '\n')

	if _, err := file.Write(encoded); err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync audit file: %w", err)
	}
	return nil
}
