package commands

import (
	"strings"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/auth"
)

func TestAuthLoginStatusLogout(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	loginOut := captureOutput(t, func() {
		cmd := NewAuthCmd()
		cmd.SetArgs([]string{"login", "--provider", "openai", "--token", "tok-openai"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("auth login execute: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(loginOut), "saved") {
		t.Fatalf("expected login output to contain saved, got: %s", loginOut)
	}

	cred, err := auth.GetCredential("openai")
	if err != nil {
		t.Fatalf("auth.GetCredential: %v", err)
	}
	if cred == nil || cred.AccessToken != "tok-openai" {
		t.Fatalf("expected stored openai token, got %+v", cred)
	}

	statusOut := captureOutput(t, func() {
		cmd := NewAuthCmd()
		cmd.SetArgs([]string{"status"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("auth status execute: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(statusOut), "openai") {
		t.Fatalf("expected status to include provider openai, got: %s", statusOut)
	}

	logoutOut := captureOutput(t, func() {
		cmd := NewAuthCmd()
		cmd.SetArgs([]string{"logout", "--provider", "openai"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("auth logout execute: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(logoutOut), "logged out") {
		t.Fatalf("expected logout output, got: %s", logoutOut)
	}

	cred, err = auth.GetCredential("openai")
	if err != nil {
		t.Fatalf("auth.GetCredential after logout: %v", err)
	}
	if cred != nil {
		t.Fatalf("expected credential removed, got %+v", cred)
	}
}

func TestAuthLoginRequiresToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cmd := NewAuthCmd()
	cmd.SetArgs([]string{"login", "--provider", "openai"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected auth login to fail without token")
	}
}

func TestAuthLoginDeviceCode(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	orig := authLoginDeviceCode
	authLoginDeviceCode = func(cfg auth.OAuthProviderConfig) (*auth.Credential, error) {
		return &auth.Credential{
			AccessToken: "oauth-device-token",
			Provider:    "openai",
			AuthMethod:  "oauth",
			ExpiresAt:   time.Now().Add(time.Hour),
		}, nil
	}
	defer func() { authLoginDeviceCode = orig }()

	cmd := NewAuthCmd()
	cmd.SetArgs([]string{"login", "--provider", "openai", "--device-code"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login --device-code execute: %v", err)
	}

	cred, err := auth.GetCredential("openai")
	if err != nil {
		t.Fatalf("auth.GetCredential: %v", err)
	}
	if cred == nil || cred.AccessToken != "oauth-device-token" {
		t.Fatalf("expected device-code credential saved, got %+v", cred)
	}
}

func TestAuthLoginBrowser(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	orig := authLoginBrowser
	authLoginBrowser = func(cfg auth.OAuthProviderConfig) (*auth.Credential, error) {
		return &auth.Credential{
			AccessToken: "oauth-browser-token",
			Provider:    "openai",
			AuthMethod:  "oauth",
			ExpiresAt:   time.Now().Add(time.Hour),
		}, nil
	}
	defer func() { authLoginBrowser = orig }()

	cmd := NewAuthCmd()
	cmd.SetArgs([]string{"login", "--provider", "openai", "--browser"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth login --browser execute: %v", err)
	}

	cred, err := auth.GetCredential("openai")
	if err != nil {
		t.Fatalf("auth.GetCredential: %v", err)
	}
	if cred == nil || cred.AccessToken != "oauth-browser-token" {
		t.Fatalf("expected browser credential saved, got %+v", cred)
	}
}
