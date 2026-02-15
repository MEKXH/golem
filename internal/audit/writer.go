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
	auditFileMode = 0644
	auditDirMode  = 0755
)

// Event is one audit record written as a single JSON line.
type Event struct {
	Time      time.Time `json:"time"`
	Type      string    `json:"type"`
	RequestID string    `json:"request_id,omitempty"`
	Tool      string    `json:"tool,omitempty"`
	Result    string    `json:"result,omitempty"`
}

// Writer appends audit events to <workspace>/state/audit.jsonl.
type Writer struct {
	path string
	mu   sync.Mutex
}

// NewWriter creates an append-only audit writer rooted at workspace state.
func NewWriter(workspace string) *Writer {
	return &Writer{
		path: filepath.Join(workspace, "state", "audit.jsonl"),
	}
}

// Append writes one event as one JSONL line.
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
