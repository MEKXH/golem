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

// HeartbeatState 存储最新的活跃聊天目标，用于心跳传递。
type HeartbeatState struct {
	LastChannel string    `json:"last_channel"`
	LastChatID  string    `json:"last_chat_id"`
	SeenAt      time.Time `json:"seen_at,omitempty"`
}

// Manager 持久化轻量级运行时状态。
type Manager struct {
	heartbeatPath string
	mu            sync.Mutex
}

// NewManager 在 <baseDir>/state 下创建状态管理器。
func NewManager(baseDir string) *Manager {
	return &Manager{
		heartbeatPath: filepath.Join(baseDir, "state", "heartbeat.json"),
	}
}

// LoadHeartbeatState 从磁盘读取心跳状态。
// 缺失或格式错误的文件将视为空状态。
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

// SaveHeartbeatState 将心跳状态写入磁盘。
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
