package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Credential 存储单个供应商的身份验证凭据（如访问令牌、刷新令牌等）。
type Credential struct {
	AccessToken  string    `json:"access_token"`            // 访问令牌
	RefreshToken string    `json:"refresh_token,omitempty"` // 刷新令牌（可选）
	Provider     string    `json:"provider"`                // 供应商名称（如 "openai"）
	AuthMethod   string    `json:"auth_method"`             // 认证方法（如 "oauth"）
	ExpiresAt    time.Time `json:"expires_at,omitempty"`    // 令牌过期时间
}

// Store 是身份验证凭据的磁盘存储容器。
type Store struct {
	Credentials map[string]*Credential `json:"credentials"` // 供应商名称到凭据的映射
}

// IsExpired 检查当前凭据是否已过期。
func (c *Credential) IsExpired() bool {
	if c == nil || c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

// NeedsRefresh 检查凭据是否即将过期（5分钟内），需要刷新。
func (c *Credential) NeedsRefresh() bool {
	if c == nil || c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(c.ExpiresAt)
}

// FilePath 返回身份验证凭据存储文件的路径 (~/.golem/auth.json)。
func FilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".golem", "auth.json")
}

// LoadStore 从磁盘加载身份验证存储。如果文件不存在，则返回一个空的存储实例。
func LoadStore() (*Store, error) {
	path := FilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{Credentials: map[string]*Credential{}}, nil
		}
		return nil, err
	}

	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Credentials == nil {
		store.Credentials = map[string]*Credential{}
	}
	return &store, nil
}

// SaveStore 将身份验证存储持久化到磁盘。
func SaveStore(store *Store) error {
	if store == nil {
		store = &Store{Credentials: map[string]*Credential{}}
	}
	if store.Credentials == nil {
		store.Credentials = map[string]*Credential{}
	}

	path := FilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

// GetCredential 获取指定供应商的身份验证凭据。
func GetCredential(provider string) (*Credential, error) {
	store, err := LoadStore()
	if err != nil {
		return nil, err
	}
	return store.Credentials[normalizeProvider(provider)], nil
}

// SetCredential 保存指定供应商的身份验证凭据到存储中。
func SetCredential(provider string, cred *Credential) error {
	store, err := LoadStore()
	if err != nil {
		return err
	}

	key := normalizeProvider(provider)
	if cred != nil {
		cred.Provider = key
	}
	store.Credentials[key] = cred
	return SaveStore(store)
}

// DeleteCredential 从存储中删除指定供应商的身份验证凭据。
func DeleteCredential(provider string) error {
	store, err := LoadStore()
	if err != nil {
		return err
	}
	delete(store.Credentials, normalizeProvider(provider))
	return SaveStore(store)
}

// DeleteAllCredentials 清空整个身份验证存储并删除相关文件。
func DeleteAllCredentials() error {
	path := FilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
