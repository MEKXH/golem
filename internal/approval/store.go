package approval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

const (
	storeVersion      = 1    // 存储文件版本
	approvalsFileMode = 0644 // 文件权限
	approvalsDirMode  = 0755 // 目录权限
	defaultStartingID = int64(1) // 默认起始 ID
)

type fileData struct {
	Version  int       `json:"version"`  // 结构版本
	NextID   int64     `json:"next_id"`  // 下一个可用 ID
	Requests []Request `json:"requests"` // 历史请求列表
}

// Store 负责将审批请求持久化到磁盘中的 JSON 文件。
type Store struct {
	path string     // 存储文件路径
	mu   sync.Mutex // 确保线程安全
}

// NewStore 在 <workspace>/state/approvals.json 创建一个审批存储实例。
func NewStore(workspace string) *Store {
	return &Store{path: filepath.Join(workspace, "state", "approvals.json")}
}

// Load 从磁盘读取持久化的审批数据。
func (s *Store) Load() (fileData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.loadLocked()
}

// Save 将审批数据写入磁盘。
func (s *Store) Save(data fileData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveLocked(data)
}

func (s *Store) loadLocked() (fileData, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultFileData(), nil
		}
		return fileData{}, fmt.Errorf("read approval store: %w", err)
	}

	var parsed fileData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fileData{}, fmt.Errorf("parse approval store: %w", err)
	}

	normalized := normalizeFileData(parsed)
	return normalized, nil
}

func (s *Store) saveLocked(data fileData) error {
	normalized := normalizeFileData(data)

	encoded, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal approval store: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), approvalsDirMode); err != nil {
		return fmt.Errorf("create approval store dir: %w", err)
	}

	dir := filepath.Dir(s.path)
	// 使用临时文件写入并重命名，以保证原子性
	tmpFile, err := os.CreateTemp(dir, "approvals-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp approval store: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(encoded); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp approval store: %w", err)
	}
	if err := tmpFile.Chmod(approvalsFileMode); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp approval store: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp approval store: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		if removeErr := os.Remove(s.path); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("replace approval store: rename failed (%v), remove failed (%v)", err, removeErr)
		}
		if retryErr := os.Rename(tmpPath, s.path); retryErr != nil {
			return fmt.Errorf("replace approval store after remove: %w", retryErr)
		}
	}
	return nil
}

func defaultFileData() fileData {
	return fileData{
		Version:  storeVersion,
		NextID:   defaultStartingID,
		Requests: []Request{},
	}
}

func normalizeFileData(data fileData) fileData {
	if data.Version <= 0 {
		data.Version = storeVersion
	}
	if data.Requests == nil {
		data.Requests = []Request{}
	}
	if data.NextID <= 0 {
		data.NextID = nextIDFromRequests(data.Requests)
	}
	return data
}

func nextIDFromRequests(requests []Request) int64 {
	maxID := int64(0)
	for _, req := range requests {
		id, err := strconv.ParseInt(req.ID, 10, 64)
		if err != nil {
			continue
		}
		if id > maxID {
			maxID = id
		}
	}
	if maxID < defaultStartingID {
		return defaultStartingID
	}
	return maxID + 1
}
