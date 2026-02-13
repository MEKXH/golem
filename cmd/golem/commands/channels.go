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
		newChannelsStartCmd(),
		newChannelsStopCmd(),
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

func newChannelsStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <channel>",
		Short: "Enable a channel in config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChannelsSetEnabled(args[0], true)
		},
	}
}

func newChannelsStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <channel>",
		Short: "Disable a channel in config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChannelsSetEnabled(args[0], false)
		},
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

	for _, state := range channelStates(cfg) {
		fmt.Printf("  %-12s %-10s %s\n", state.Name, state.Status(), state.Note())
	}

	return nil
}

func runChannelsStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("=== Channel Status ===")
	fmt.Println()

	for _, state := range channelStates(cfg) {
		fmt.Printf("%s:\n", titleCase(state.Name))
		fmt.Printf("  Enabled:    %v\n", state.Enabled)
		fmt.Printf("  Readiness:  %s\n", state.Note())
		if len(state.AllowFrom) > 0 {
			fmt.Printf("  Allow From: %s\n", strings.Join(state.AllowFrom, ", "))
		} else {
			fmt.Println("  Allow From: all (no restrictions)")
		}
		fmt.Println()
	}

	return nil
}

func runChannelsSetEnabled(channelName string, enabled bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := strings.ToLower(strings.TrimSpace(channelName))
	switch name {
	case "telegram":
		cfg.Channels.Telegram.Enabled = enabled
	case "whatsapp":
		cfg.Channels.WhatsApp.Enabled = enabled
	case "feishu":
		cfg.Channels.Feishu.Enabled = enabled
	case "discord":
		cfg.Channels.Discord.Enabled = enabled
	case "slack":
		cfg.Channels.Slack.Enabled = enabled
	case "qq":
		cfg.Channels.QQ.Enabled = enabled
	case "dingtalk":
		cfg.Channels.DingTalk.Enabled = enabled
	case "maixcam":
		cfg.Channels.MaixCam.Enabled = enabled
	default:
		return fmt.Errorf("unknown channel: %s", channelName)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	state := "disabled"
	if enabled {
		state = "enabled"
	}
	fmt.Printf("Channel %s %s.\n", name, state)
	return nil
}

type channelState struct {
	Name      string
	Enabled   bool
	Ready     bool
	Reason    string
	AllowFrom []string
}

func (s channelState) Status() string {
	if s.Enabled {
		return "enabled"
	}
	return "disabled"
}

func (s channelState) Note() string {
	if !s.Enabled {
		return ""
	}
	if s.Ready {
		return "ready"
	}
	return s.Reason
}

func channelStates(cfg *config.Config) []channelState {
	return []channelState{
		{
			Name:      "telegram",
			Enabled:   cfg.Channels.Telegram.Enabled,
			Ready:     strings.TrimSpace(cfg.Channels.Telegram.Token) != "",
			Reason:    "token not set",
			AllowFrom: cfg.Channels.Telegram.AllowFrom,
		},
		{
			Name:      "whatsapp",
			Enabled:   cfg.Channels.WhatsApp.Enabled,
			Ready:     strings.TrimSpace(cfg.Channels.WhatsApp.BridgeURL) != "",
			Reason:    "bridge_url not set",
			AllowFrom: cfg.Channels.WhatsApp.AllowFrom,
		},
		{
			Name:      "feishu",
			Enabled:   cfg.Channels.Feishu.Enabled,
			Ready:     strings.TrimSpace(cfg.Channels.Feishu.AppID) != "" && strings.TrimSpace(cfg.Channels.Feishu.AppSecret) != "",
			Reason:    "app_id/app_secret not set",
			AllowFrom: cfg.Channels.Feishu.AllowFrom,
		},
		{
			Name:      "discord",
			Enabled:   cfg.Channels.Discord.Enabled,
			Ready:     strings.TrimSpace(cfg.Channels.Discord.Token) != "",
			Reason:    "token not set",
			AllowFrom: cfg.Channels.Discord.AllowFrom,
		},
		{
			Name:      "slack",
			Enabled:   cfg.Channels.Slack.Enabled,
			Ready:     strings.TrimSpace(cfg.Channels.Slack.BotToken) != "" && strings.TrimSpace(cfg.Channels.Slack.AppToken) != "",
			Reason:    "bot_token/app_token not set",
			AllowFrom: cfg.Channels.Slack.AllowFrom,
		},
		{
			Name:      "qq",
			Enabled:   cfg.Channels.QQ.Enabled,
			Ready:     strings.TrimSpace(cfg.Channels.QQ.AppID) != "" && strings.TrimSpace(cfg.Channels.QQ.AppSecret) != "",
			Reason:    "app_id/app_secret not set",
			AllowFrom: cfg.Channels.QQ.AllowFrom,
		},
		{
			Name:      "dingtalk",
			Enabled:   cfg.Channels.DingTalk.Enabled,
			Ready:     strings.TrimSpace(cfg.Channels.DingTalk.ClientID) != "" && strings.TrimSpace(cfg.Channels.DingTalk.ClientSecret) != "",
			Reason:    "client_id/client_secret not set",
			AllowFrom: cfg.Channels.DingTalk.AllowFrom,
		},
		{
			Name:      "maixcam",
			Enabled:   cfg.Channels.MaixCam.Enabled,
			Ready:     strings.TrimSpace(cfg.Channels.MaixCam.Host) != "" && cfg.Channels.MaixCam.Port > 0,
			Reason:    "host/port not set",
			AllowFrom: cfg.Channels.MaixCam.AllowFrom,
		},
	}
}

func titleCase(name string) string {
	switch strings.ToLower(name) {
	case "qq":
		return "QQ"
	case "maixcam":
		return "MaixCam"
	case "dingtalk":
		return "DingTalk"
	}
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
