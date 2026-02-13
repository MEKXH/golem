package commands

import (
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/config"
	"github.com/spf13/cobra"
)

func NewChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "Manage communication channels",
	}

	cmd.AddCommand(
		newChannelsListCmd(),
		newChannelsStatusCmd(),
	)

	return cmd
}

func newChannelsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured channels",
		RunE:  runChannelsList,
	}
}

func newChannelsStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show detailed channel status",
		RunE:  runChannelsStatus,
	}
}

func runChannelsList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Channels:")
	fmt.Printf("  %-12s %-10s %s\n", "NAME", "STATUS", "NOTE")
	fmt.Printf("  %-12s %-10s %s\n", strings.Repeat("-", 12), strings.Repeat("-", 10), strings.Repeat("-", 20))

	// Telegram
	tgStatus := "disabled"
	tgNote := ""
	if cfg.Channels.Telegram.Enabled {
		tgStatus = "enabled"
		if strings.TrimSpace(cfg.Channels.Telegram.Token) == "" {
			tgNote = "token not set"
		} else {
			tgNote = "ready"
		}
	}
	fmt.Printf("  %-12s %-10s %s\n", "telegram", tgStatus, tgNote)

	return nil
}

func runChannelsStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("=== Channel Status ===")
	fmt.Println()

	// Telegram
	fmt.Println("Telegram:")
	fmt.Printf("  Enabled:    %v\n", cfg.Channels.Telegram.Enabled)
	if strings.TrimSpace(cfg.Channels.Telegram.Token) != "" {
		fmt.Println("  Token:      configured")
	} else {
		fmt.Println("  Token:      not set")
	}
	if len(cfg.Channels.Telegram.AllowFrom) > 0 {
		fmt.Printf("  Allow From: %s\n", strings.Join(cfg.Channels.Telegram.AllowFrom, ", "))
	} else {
		fmt.Println("  Allow From: all (no restrictions)")
	}

	readiness := "not ready"
	if cfg.Channels.Telegram.Enabled && strings.TrimSpace(cfg.Channels.Telegram.Token) != "" {
		readiness = "ready"
	}
	fmt.Printf("  Readiness:  %s\n", readiness)

	return nil
}
