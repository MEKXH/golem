package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/auth"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	authLoginBrowser    = auth.LoginBrowser
	authLoginDeviceCode = auth.LoginDeviceCode
)

// NewAuthCmd 创建身份认证管理命令。
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
				fmt.Println("No authenticated providers. Use 'golem auth login --provider <name>' to authenticate.")
				return nil
			}

			providers := make([]string, 0, len(store.Credentials))
			for provider := range store.Credentials {
				providers = append(providers, provider)
			}
			sort.Strings(providers)

			var (
				wProvider = 15
				wMethod   = 12
				wStatus   = 15
				wExpires  = 25

				colHeaderStyle = lipgloss.NewStyle().
						Foreground(lipgloss.Color("#8E4EC6")). // Purple
						Bold(true).
						MarginRight(1)

				providerStyleBase = lipgloss.NewStyle().
							Width(wProvider).
							MarginRight(1)

				methodStyleBase = lipgloss.NewStyle().
						Width(wMethod).
						MarginRight(1)

				statusStyleBase = lipgloss.NewStyle().
						Width(wStatus).
						MarginRight(1)

				expiresStyleBase = lipgloss.NewStyle().
							Width(wExpires).
							MarginRight(1)

				okColor      = lipgloss.Color("#2E8B57") // SeaGreen
				warnColor    = lipgloss.Color("#FFA500") // Orange
				errorColor   = lipgloss.Color("#FF0000") // Red
				defaultColor = lipgloss.Color("241")
			)

			fmt.Println("Authenticated providers:")
			fmt.Println()

			headers := lipgloss.JoinHorizontal(lipgloss.Top,
				colHeaderStyle.Width(wProvider).Render("PROVIDER"),
				colHeaderStyle.Width(wMethod).Render("METHOD"),
				colHeaderStyle.Width(wStatus).Render("STATUS"),
				colHeaderStyle.Width(wExpires).Render("EXPIRES"),
			)
			fmt.Printf("  %s\n", headers)

			sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginRight(1)
			separator := lipgloss.JoinHorizontal(lipgloss.Top,
				sepStyle.Render(strings.Repeat("─", wProvider)),
				sepStyle.Render(strings.Repeat("─", wMethod)),
				sepStyle.Render(strings.Repeat("─", wStatus)),
				sepStyle.Render(strings.Repeat("─", wExpires)),
			)
			fmt.Printf("  %s\n", separator)

			for _, p := range providers {
				cred := store.Credentials[p]
				status := "active"
				sColor := okColor

				if cred.IsExpired() {
					status = "expired"
					sColor = errorColor
				} else if cred.NeedsRefresh() {
					status = "needs_refresh"
					sColor = warnColor
				}

				expires := "-"
				if !cred.ExpiresAt.IsZero() {
					expires = cred.ExpiresAt.Format(time.RFC3339)
				}

				row := lipgloss.JoinHorizontal(lipgloss.Top,
					providerStyleBase.Render(truncate(p, wProvider)),
					methodStyleBase.Foreground(defaultColor).Render(cred.AuthMethod),
					statusStyleBase.Foreground(sColor).Render(status),
					expiresStyleBase.Foreground(defaultColor).Render(expires),
				)
				fmt.Printf("  %s\n", row)
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
