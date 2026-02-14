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

// HeartbeatState stores the latest active chat target for heartbeat delivery.
type HeartbeatState struct {
	LastChannel string    `json:"last_channel"`
	LastChatID  string    `json:"last_chat_id"`
	SeenAt      time.Time `json:"seen_at,omitempty"`
}

// Manager persists lightweight runtime state.
type Manager struct {
	heartbeatPath string
	mu            sync.Mutex
}

// NewManager creates a state manager under <baseDir>/state.
func NewManager(baseDir string) *Manager {
	return &Manager{
		heartbeatPath: filepath.Join(baseDir, "state", "heartbeat.json"),
	}
}

// LoadHeartbeatState reads heartbeat state from disk.
// Missing or malformed files are treated as empty state.
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

// SaveHeartbeatState writes heartbeat state to disk.
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
