package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/auth"
	"github.com/spf13/cobra"
)

var (
	authLoginBrowser    = auth.LoginBrowser
	authLoginDeviceCode = auth.LoginDeviceCode
)

func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage provider authentication tokens",
	}

	cmd.AddCommand(
		newAuthLoginCmd(),
		newAuthLogoutCmd(),
		newAuthStatusCmd(),
	)

	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var provider string
	var token string
	var browser bool
	var deviceCode bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login with token or OAuth device/browser flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := normalizeAuthProvider(provider)
			if err != nil {
				return err
			}
			if browser && deviceCode {
				return fmt.Errorf("choose only one oauth flow: --browser or --device-code")
			}

			var cred *auth.Credential
			if browser || deviceCode {
				cfg, err := oauthConfigForProvider(p)
				if err != nil {
					return err
				}
				if browser {
					cred, err = authLoginBrowser(cfg)
				} else {
					cred, err = authLoginDeviceCode(cfg)
				}
				if err != nil {
					return fmt.Errorf("oauth login failed: %w", err)
				}
			} else {
				token = strings.TrimSpace(token)
				if token == "" {
					return fmt.Errorf("token is required for login")
				}
				cred = &auth.Credential{
					AccessToken: token,
					Provider:    p,
					AuthMethod:  "token",
				}
			}

			if cred == nil || strings.TrimSpace(cred.AccessToken) == "" {
				return fmt.Errorf("login returned empty credential")
			}
			cred.Provider = p
			if strings.TrimSpace(cred.AuthMethod) == "" {
				cred.AuthMethod = "token"
			}

			if err := auth.SetCredential(p, cred); err != nil {
				return fmt.Errorf("save credential: %w", err)
			}

			fmt.Printf("Credential saved for %s (method=%s).\n", p, cred.AuthMethod)
			return nil
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider name (openai|claude|openrouter|deepseek|gemini|ark|qianfan|qwen)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Access token to save")
	cmd.Flags().BoolVar(&browser, "browser", false, "Use OAuth browser flow")
	cmd.Flags().BoolVar(&deviceCode, "device-code", false, "Use OAuth device-code flow")
	_ = cmd.MarkFlagRequired("provider")

	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove one provider token or all tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := strings.TrimSpace(provider)
			if p == "" {
				if err := auth.DeleteAllCredentials(); err != nil {
					return fmt.Errorf("delete all credentials: %w", err)
				}
				fmt.Println("Logged out from all providers.")
				return nil
			}

			norm, err := normalizeAuthProvider(p)
			if err != nil {
				return err
			}
			if err := auth.DeleteCredential(norm); err != nil {
				return fmt.Errorf("delete credential: %w", err)
			}
			fmt.Printf("Logged out from %s.\n", norm)
			return nil
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider name")
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current auth credential status",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := auth.LoadStore()
			if err != nil {
				return fmt.Errorf("load auth store: %w", err)
			}
			if len(store.Credentials) == 0 {
				fmt.Println("No authenticated providers.")
				return nil
			}

			providers := make([]string, 0, len(store.Credentials))
			for provider := range store.Credentials {
				providers = append(providers, provider)
			}
			sort.Strings(providers)

			fmt.Println("Authenticated providers:")
			for _, p := range providers {
				cred := store.Credentials[p]
				status := "active"
				if cred.IsExpired() {
					status = "expired"
				} else if cred.NeedsRefresh() {
					status = "needs_refresh"
				}

				expires := ""
				if !cred.ExpiresAt.IsZero() {
					expires = " expires=" + cred.ExpiresAt.Format(time.RFC3339)
				}
				fmt.Printf("- %s method=%s status=%s%s\n", p, cred.AuthMethod, status, expires)
			}
			return nil
		},
	}
}

func normalizeAuthProvider(provider string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "openai", "claude", "openrouter", "deepseek", "gemini", "ark", "qianfan", "qwen":
		return p, nil
	case "anthropic":
		return "claude", nil
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func oauthConfigForProvider(provider string) (auth.OAuthProviderConfig, error) {
	switch provider {
	case "openai":
		return auth.OpenAIOAuthConfig(), nil
	default:
		return auth.OAuthProviderConfig{}, fmt.Errorf("oauth is not supported for provider: %s", provider)
	}
}
