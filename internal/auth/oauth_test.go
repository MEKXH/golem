package auth

import "testing"

func TestOpenAIOAuthConfig(t *testing.T) {
	cfg := OpenAIOAuthConfig()
	if cfg.Issuer == "" || cfg.ClientID == "" {
		t.Fatalf("invalid OpenAIOAuthConfig: %+v", cfg)
	}
}

func TestBuildAuthorizeURL(t *testing.T) {
	cfg := OAuthProviderConfig{
		Issuer:   "https://auth.example.com",
		ClientID: "client-1",
		Scopes:   "openid profile",
		Port:     1455,
	}
	pkce := PKCECodes{
		CodeVerifier:  "verifier",
		CodeChallenge: "challenge",
	}

	url := BuildAuthorizeURL(cfg, pkce, "state-1", "http://localhost:1455/auth/callback")
	if url == "" {
		t.Fatal("expected authorize url")
	}
}
