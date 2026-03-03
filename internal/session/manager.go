package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Message 表示会话中的单条消息
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Session 表示一个会话
type Session struct {
	Key      string
	Messages []*Message
	mu       sync.RWMutex
}

// AddMessage 向会话添加一条消息
func (s *Session) AddMessage(role, content string) *Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := &Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
	s.Messages = append(s.Messages, msg)
	return msg
}

// GetHistory 返回最近 n 条消息
func (s *Session) GetHistory(limit int) []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.Messages) {
		limit = len(s.Messages)
	}
	start := len(s.Messages) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*Message, limit)
	copy(result, s.Messages[start:])
	return result
}

// Manager 管理会话
type Manager struct {
	dir      string
	sessions map[string]*Session
	mu       sync.RWMutex
}

const maxSessionLineBytes = 4 * 1024 * 1024

// NewManager 创建一个会话管理器
func NewManager(baseDir string) *Manager {
	dir := filepath.Join(baseDir, "sessions")
	os.MkdirAll(dir, 0755)
	return &Manager{
		dir:      dir,
		sessions: make(map[string]*Session),
	}
}

// GetOrCreate 获取或创建一个会话
func (m *Manager) GetOrCreate(key string) *Session {
	m.mu.RLock()
	if sess, ok := m.sessions[key]; ok {
		m.mu.RUnlock()
		return sess
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after acquiring write lock
	if sess, ok := m.sessions[key]; ok {
		return sess
	}

	sess := &Session{Key: key}
	if err := m.loadFromDisk(sess); err != nil {
		slog.Warn("failed to load session from disk", "session_key", key, "error", err)
	}
	m.sessions[key] = sess
	return sess
}

// Save 将会话持久化到磁盘
func (m *Manager) Save(sess *Session) error {
	sess.mu.RLock()
	defer sess.mu.RUnlock()

	if len(sess.Messages) == 0 {
		return nil
	}

	path := m.sessionPath(sess.Key)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, msg := range sess.Messages {
		if err := enc.Encode(msg); err != nil {
			return err
		}
	}
	return nil
}

// Append 追加消息到会话文件
func (m *Manager) Append(key string, msgs ...*Message) error {
	path := m.sessionPath(key)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, msg := range msgs {
		if err := enc.Encode(msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) loadFromDisk(sess *Session) error {
	path := m.sessionPath(sess.Key)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSessionLineBytes)
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil {
			sess.Messages = append(sess.Messages, &msg)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan session file %s: %w", path, err)
	}
	return nil
}

// Reset 清除会话在内存中的历史记录并从磁盘删除其文件。
func (m *Manager) Reset(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[key]; ok {
		sess.mu.Lock()
		sess.Messages = nil
		sess.mu.Unlock()
	}
	os.Remove(m.sessionPath(key))
}

func (m *Manager) sessionPath(key string) string {
	safeKey := strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(key)
	return filepath.Join(m.dir, safeKey+".jsonl")
}
