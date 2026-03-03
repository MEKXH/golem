// Package state 提供轻量级的运行时状态持久化管理。
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const heartbeatStateFileMode = 0600

// HeartbeatState 存储最新的活跃聊天目标，用于心跳消息的定向投递。
type HeartbeatState struct {
	LastChannel string    `json:"last_channel"` // 最近活跃的消息通道
	LastChatID  string    `json:"last_chat_id"`  // 最近活跃的聊天 ID
	SeenAt      time.Time `json:"seen_at,omitempty"` // 最近一次活跃的时间戳
}

// Manager 负责在磁盘上持久化轻量级的运行时状态。
type Manager struct {
	heartbeatPath string     // 心跳状态文件的存储路径
	mu            sync.Mutex // 确保文件读写的线程安全
}

// NewManager 在指定的基础目录下创建状态管理器，文件将存储在 <baseDir>/state 中。
func NewManager(baseDir string) *Manager {
	return &Manager{
		heartbeatPath: filepath.Join(baseDir, "state", "heartbeat.json"),
	}
}

// LoadHeartbeatState 从磁盘加载心跳状态。如果文件不存在或格式错误，将返回空状态。
func (m *Manager) LoadHeartbeatState() (HeartbeatState, error) {
	data, err := os.ReadFile(m.heartbeatPath)
	if err != nil {
		if os.IsNotExist(err) {
			return HeartbeatState{}, nil
		}
		return HeartbeatState{}, err
	}

	var st HeartbeatState
	if err := json.Unmarshal(data, &st); err != nil {
		return HeartbeatState{}, nil
	}
	st.LastChannel = strings.TrimSpace(st.LastChannel)
	st.LastChatID = strings.TrimSpace(st.LastChatID)
	if st.LastChannel == "" || st.LastChatID == "" {
		return HeartbeatState{}, nil
	}
	return st, nil
}

// SaveHeartbeatState 将最新的心跳状态写入磁盘。
func (m *Manager) SaveHeartbeatState(st HeartbeatState) error {
	st.LastChannel = strings.TrimSpace(st.LastChannel)
	st.LastChatID = strings.TrimSpace(st.LastChatID)
	if st.LastChannel == "" || st.LastChatID == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(m.heartbeatPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.heartbeatPath, data, heartbeatStateFileMode)
}
