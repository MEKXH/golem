package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// OAuthProviderConfig defines OAuth metadata for one provider.
type OAuthProviderConfig struct {
	Issuer   string
	ClientID string
	Scopes   string
	Port     int
}

// OpenAIOAuthConfig returns OpenAI OAuth defaults.
func OpenAIOAuthConfig() OAuthProviderConfig {
	return OAuthProviderConfig{
		Issuer:   "https://auth.openai.com",
		ClientID: "app_EMoamEEZ73f0CkXaXp7hrann",
		Scopes:   "openid profile email offline_access",
		Port:     1455,
	}
}

func generateState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

type callbackResult struct {
	code string
	err  error
}

// LoginBrowser completes OAuth login via local callback HTTP server.
func LoginBrowser(cfg OAuthProviderConfig) (*Credential, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate pkce: %w", err)
	}
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/auth/callback", cfg.Port)
	authURL := BuildAuthorizeURL(cfg, pkce, state, redirectURI)
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			resultCh <- callbackResult{err: fmt.Errorf("state mismatch")}
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			resultCh <- callbackResult{err: fmt.Errorf("no authorization code received")}
			http.Error(w, "no code received", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body><h3>Login successful. You can close this tab.</h3></body></html>"))
		resultCh <- callbackResult{code: code}
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("listen callback port %d: %w", cfg.Port, err)
	}

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	fmt.Printf("Open this URL to authenticate:\n\n%s\n\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}
		return exchangeCodeForTokens(cfg, result.code, pkce.CodeVerifier, redirectURI)
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timed out")
	}
}

type deviceCodeResponse struct {
	DeviceAuthID string
	UserCode     string
	Interval     int
}

func parseDeviceCodeResponse(body []byte) (deviceCodeResponse, error) {
	var raw struct {
		DeviceAuthID string          `json:"device_auth_id"`
		UserCode     string          `json:"user_code"`
		Interval     json.RawMessage `json:"interval"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return deviceCodeResponse{}, err
	}

	interval, err := parseFlexibleInt(raw.Interval)
	if err != nil {
		return deviceCodeResponse{}, err
	}
	if interval < 1 {
		interval = 5
	}

	return deviceCodeResponse{
		DeviceAuthID: raw.DeviceAuthID,
		UserCode:     raw.UserCode,
		Interval:     interval,
	}, nil
}

func parseFlexibleInt(raw json.RawMessage) (int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}

	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, nil
		}
		return strconv.Atoi(s)
	}

	return 0, fmt.Errorf("invalid integer value: %s", string(raw))
}

// LoginDeviceCode completes OAuth login via device-code flow.
func LoginDeviceCode(cfg OAuthProviderConfig) (*Credential, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"client_id": cfg.ClientID,
	})

	resp, err := http.Post(
		cfg.Issuer+"/api/accounts/deviceauth/usercode",
		"application/json",
		strings.NewReader(string(reqBody)),
	)
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed: %s", string(body))
	}

	deviceResp, err := parseDeviceCodeResponse(body)
	if err != nil {
		return nil, fmt.Errorf("parse device code response: %w", err)
	}

	fmt.Printf("\nOpen this URL in browser:\n  %s/codex/device\nThen enter code: %s\n", cfg.Issuer, deviceResp.UserCode)
	fmt.Println("Waiting for authentication...")

	deadline := time.After(15 * time.Minute)
	ticker := time.NewTicker(time.Duration(deviceResp.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return nil, fmt.Errorf("device code authentication timed out")
		case <-ticker.C:
			cred, err := pollDeviceCode(cfg, deviceResp.DeviceAuthID, deviceResp.UserCode)
			if err != nil {
				continue
			}
			if cred != nil {
				return cred, nil
			}
		}
	}
}

func pollDeviceCode(cfg OAuthProviderConfig, deviceAuthID, userCode string) (*Credential, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"device_auth_id": deviceAuthID,
		"user_code":      userCode,
	})

	resp, err := http.Post(
		cfg.Issuer+"/api/accounts/deviceauth/token",
		"application/json",
		strings.NewReader(string(reqBody)),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pending")
	}

	body, _ := io.ReadAll(resp.Body)
	var tokenResp struct {
		AuthorizationCode string `json:"authorization_code"`
		CodeVerifier      string `json:"code_verifier"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	redirectURI := cfg.Issuer + "/deviceauth/callback"
	return exchangeCodeForTokens(cfg, tokenResp.AuthorizationCode, tokenResp.CodeVerifier, redirectURI)
}

// RefreshAccessToken refreshes OAuth access token when refresh token exists.
func RefreshAccessToken(cred *Credential, cfg OAuthProviderConfig) (*Credential, error) {
	if cred == nil || strings.TrimSpace(cred.RefreshToken) == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	data := url.Values{
		"client_id":     {cfg.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {cred.RefreshToken},
		"scope":         {"openid profile email"},
	}

	resp, err := http.PostForm(cfg.Issuer+"/oauth/token", data)
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	refreshed, err := parseTokenResponse(body, cred.Provider)
	if err != nil {
		return nil, err
	}
	if refreshed.Provider == "" {
		refreshed.Provider = cred.Provider
	}
	return refreshed, nil
}

// BuildAuthorizeURL builds OAuth authorize URL with PKCE.
func BuildAuthorizeURL(cfg OAuthProviderConfig, pkce PKCECodes, state, redirectURI string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {cfg.Scopes},
		"code_challenge":        {pkce.CodeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}
	return cfg.Issuer + "/authorize?" + params.Encode()
}

func exchangeCodeForTokens(cfg OAuthProviderConfig, code, codeVerifier, redirectURI string) (*Credential, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {cfg.ClientID},
		"code_verifier": {codeVerifier},
	}

	resp, err := http.PostForm(cfg.Issuer+"/oauth/token", data)
	if err != nil {
		return nil, fmt.Errorf("exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	return parseTokenResponse(body, "openai")
}

func parseTokenResponse(body []byte, provider string) (*Credential, error) {
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return &Credential{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		Provider:     provider,
		AuthMethod:   "oauth",
		ExpiresAt:    expiresAt,
	}, nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
