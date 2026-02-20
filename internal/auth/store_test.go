package auth

import (
	"testing"
	"time"
)

func TestSetGetCredential(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cred := &Credential{
		AccessToken:  "tok-openai",
		RefreshToken: "refresh-openai",
		Provider:     "openai",
		AuthMethod:   "token",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}

	if err := SetCredential("openai", cred); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}

	got, err := GetCredential("openai")
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if got == nil {
		t.Fatal("expected credential, got nil")
	}
	if got.AccessToken != cred.AccessToken {
		t.Fatalf("expected access token %q, got %q", cred.AccessToken, got.AccessToken)
	}
	if got.Provider != "openai" {
		t.Fatalf("expected provider openai, got %q", got.Provider)
	}
}

func TestDeleteCredential(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := SetCredential("openai", &Credential{
		AccessToken: "tok-openai",
		Provider:    "openai",
		AuthMethod:  "token",
	}); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}

	if err := DeleteCredential("openai"); err != nil {
		t.Fatalf("DeleteCredential: %v", err)
	}

	got, err := GetCredential("openai")
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil credential after delete, got %+v", got)
	}
}

func TestDeleteAllCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := SetCredential("openai", &Credential{
		AccessToken: "tok-openai",
		Provider:    "openai",
		AuthMethod:  "token",
	}); err != nil {
		t.Fatalf("SetCredential openai: %v", err)
	}
	if err := SetCredential("claude", &Credential{
		AccessToken: "tok-claude",
		Provider:    "claude",
		AuthMethod:  "token",
	}); err != nil {
		t.Fatalf("SetCredential claude: %v", err)
	}

	if err := DeleteAllCredentials(); err != nil {
		t.Fatalf("DeleteAllCredentials: %v", err)
	}

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore: %v", err)
	}
	if len(store.Credentials) != 0 {
		t.Fatalf("expected empty credentials, got %d", len(store.Credentials))
	}
}
