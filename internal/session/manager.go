// Package session 实现会话管理功能，用于存储和检索用户与 Agent 之间的聊天历史记录。
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

// Message 表示会话中的单条消息记录。
type Message struct {
	Role      string    `json:"role"`      // 角色：user 或 assistant
	Content   string    `json:"content"`   // 消息文本内容
	Timestamp time.Time `json:"timestamp"` // 消息产生的时间戳
}

// Session 表示一个完整的对话会话，包含唯一的标识符和消息序列。
type Session struct {
	Key      string       // 会话的唯一键值
	Messages []*Message   // 消息历史列表
	mu       sync.RWMutex // 保护 Messages 列表的并发安全
}

// AddMessage 向会话中追加一条新消息。
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

// GetHistory 返回会话中最近的 n 条消息。
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

// Manager 负责管理内存中的活跃会话，并将其持久化到磁盘。
type Manager struct {
	dir      string              // 会话文件存储目录
	sessions map[string]*Session // 内存缓存的会话映射
	mu       sync.RWMutex
}

const maxSessionLineBytes = 4 * 1024 * 1024 // 单行消息的最大字节数 (4MB)

// NewManager 在指定的基础目录下创建一个新的会话管理器。
func NewManager(baseDir string) *Manager {
	dir := filepath.Join(baseDir, "sessions")
	os.MkdirAll(dir, 0755)
	return &Manager{
		dir:      dir,
		sessions: make(map[string]*Session),
	}
}

// GetOrCreate 获取指定键值的会话实例。如果内存中不存在，则尝试从磁盘加载或创建一个新会话。
func (m *Manager) GetOrCreate(key string) *Session {
	m.mu.RLock()
	if sess, ok := m.sessions[key]; ok {
		m.mu.RUnlock()
		return sess
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查锁定
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

// Save 将指定的会话内容完整覆盖写入到磁盘文件中。
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

// Append 将一组新消息增量追加到指定的会话持久化文件中。
func (m *Manager) Append(key string, msgs ...*Message) error {
	path := m.sessionPath(key)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Use a buffered writer to minimize disk I/O syscalls during multiple small appends.
	bw := bufio.NewWriter(f)
	enc := json.NewEncoder(bw)
	for _, msg := range msgs {
		if err := enc.Encode(msg); err != nil {
			return err
		}
	}
	if err := bw.Flush(); err != nil {
		return err
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

// Reset 清除会话在内存中的历史记录，并从磁盘中永久删除对应的会话文件。
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

// sessionPathReplacer is cached globally to avoid O(N) allocation and
// initialization overhead of strings.NewReplacer on every file I/O operation.
var sessionPathReplacer = strings.NewReplacer(":", "_", "/", "_", "\\", "_")

func (m *Manager) sessionPath(key string) string {
	// 转换键值中的敏感字符以生成安全的文件名
	safeKey := sessionPathReplacer.Replace(key)
	return filepath.Join(m.dir, safeKey+".jsonl")
}
