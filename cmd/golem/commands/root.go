// Package commands 提供 Golem CLI 的各个子命令实现。
package commands

import (
	"strings"

	"github.com/MEKXH/golem/internal/config"
	"github.com/spf13/cobra"
)

// logLevelOverride 用于存储通过命令行标志覆盖的日志级别。
var logLevelOverride string

// NewRootCmd 创建并配置 Golem 的根命令行对象。
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "golem",
		Short: "Golem - Lightweight AI Assistant",
		Long:  `Golem is a lightweight personal AI assistant built with Go and Eino.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if shouldUseDefaultLogger(cmd) {
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

	// 注册所有子命令
	cmd.AddCommand(
		NewInitCmd(),
		NewChatCmd(),
		NewRunCmd(),
		NewStatusCmd(),
		NewPolicyCmd(),
		NewMCPCmd(),
		NewChannelsCmd(),
		NewApprovalCmd(),
		NewCronCmd(),
		NewSkillsCmd(),
		NewAuthCmd(),
		NewVersionCmd(),
	)

	return cmd
}

// shouldUseDefaultLogger 判断当前执行的命令是否应当使用默认日志配置（如初始化或认证相关命令）。
func shouldUseDefaultLogger(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if cmd.Name() == "init" {
		return true
	}
	return strings.HasPrefix(cmd.CommandPath(), "golem auth")
}
