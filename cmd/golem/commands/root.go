package commands

import (
	"github.com/MEKXH/golem/internal/config"
	"github.com/spf13/cobra"
)

var logLevelOverride string

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "golem",
		Short: "Golem - Lightweight AI Assistant",
		Long:  `Golem is a lightweight personal AI assistant built with Go and Eino.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "init" {
				return configureLogger(config.DefaultConfig(), logLevelOverride, false)
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return configureLogger(cfg, logLevelOverride, cmd.Name() == "chat")
		},
	}

	cmd.PersistentFlags().StringVar(&logLevelOverride, "log-level", "", "Override log level (debug|info|warn|error)")

	cmd.AddCommand(
		NewInitCmd(),
		NewChatCmd(),
		NewRunCmd(),
		NewStatusCmd(),
	)

	return cmd
}
