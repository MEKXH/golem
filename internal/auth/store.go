package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Credential stores auth tokens for one provider.
type Credential struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Provider     string    `json:"provider"`
	AuthMethod   string    `json:"auth_method"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// Store is the on-disk auth container.
type Store struct {
	Credentials map[string]*Credential `json:"credentials"`
}

func (c *Credential) IsExpired() bool {
	if c == nil || c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

func (c *Credential) NeedsRefresh() bool {
	if c == nil || c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(c.ExpiresAt)
}

// FilePath returns ~/.golem/auth.json.
func FilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".golem", "auth.json")
}

// LoadStore loads auth store from disk.
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

// SaveStore persists auth store to disk.
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

// GetCredential retrieves one provider credential.
func GetCredential(provider string) (*Credential, error) {
	store, err := LoadStore()
	if err != nil {
		return nil, err
	}
	return store.Credentials[normalizeProvider(provider)], nil
}

// SetCredential saves one provider credential.
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

// DeleteCredential removes one provider credential.
func DeleteCredential(provider string) error {
	store, err := LoadStore()
	if err != nil {
		return err
	}
	delete(store.Credentials, normalizeProvider(provider))
	return SaveStore(store)
}

// DeleteAllCredentials clears auth store.
func DeleteAllCredentials() error {
	path := FilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
