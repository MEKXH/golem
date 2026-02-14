package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/MEKXH/golem/internal/agent"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/channel"
	"github.com/MEKXH/golem/internal/channel/dingtalk"
	"github.com/MEKXH/golem/internal/channel/discord"
	"github.com/MEKXH/golem/internal/channel/feishu"
	"github.com/MEKXH/golem/internal/channel/maixcam"
	"github.com/MEKXH/golem/internal/channel/qq"
	"github.com/MEKXH/golem/internal/channel/slack"
	"github.com/MEKXH/golem/internal/channel/telegram"
	"github.com/MEKXH/golem/internal/channel/whatsapp"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
	"github.com/MEKXH/golem/internal/gateway"
	"github.com/MEKXH/golem/internal/heartbeat"
	"github.com/MEKXH/golem/internal/provider"
	"github.com/MEKXH/golem/internal/tools"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start Golem server",
		RunE:  runServer,
	}

	return cmd
}

func runServer(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	msgBus := bus.NewMessageBus(100)

	model, err := provider.NewChatModel(ctx, cfg)
	if err != nil {
		slog.Warn("no model configured", "error", err)
	}

	loop, err := agent.NewLoop(cfg, msgBus, model)
	if err != nil {
		return fmt.Errorf("invalid workspace: %w", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		return err
	}

	// Initialize cron service.
	workspacePath, _ := cfg.WorkspacePathChecked()
	cronStorePath := filepath.Join(workspacePath, "cron", "jobs.json")
	cronService := cron.NewService(cronStorePath, func(job *cron.Job) error {
		ch := job.Payload.Channel
		if ch == "" {
			ch = "cron"
		}
		chatID := job.Payload.ChatID
		if chatID == "" {
			chatID = "default"
		}
		_, err := loop.ProcessForChannel(ctx, ch, chatID, "cron", job.Payload.Message)
		return err
	})
	cronTool, err := tools.NewCronTool(cronService)
	if err != nil {
		return fmt.Errorf("failed to create cron tool: %w", err)
	}
	if err := loop.Tools().Register(cronTool); err != nil {
		return fmt.Errorf("failed to register cron tool: %w", err)
	}
	if err := cronService.Start(); err != nil {
		slog.Warn("cron service failed to start", "error", err)
	}

	heartbeatService := buildHeartbeatService(cfg, msgBus, cronService)
	loop.SetActivityRecorder(heartbeatService.TrackActivity)
	if err := heartbeatService.Start(); err != nil {
		slog.Warn("heartbeat service failed to start", "error", err)
	}

	errCh := make(chan error, 2)
	go func() {
		if err := loop.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("agent loop failed: %w", err)
		}
	}()

	chanMgr := channel.NewManager(msgBus)
	registerEnabledChannels(cfg, msgBus, chanMgr)

	chanMgr.StartAll(ctx)
	go chanMgr.RouteOutbound(ctx)

	gatewayServer := gateway.New(cfg.Gateway, loop)
	go func() {
		if err := gatewayServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("gateway server failed: %w", err)
		}
	}()

	fmt.Printf("Golem server running. Gateway: http://%s\nPress Ctrl+C to stop.\n", gatewayServer.Addr())

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-errCh:
		slog.Error("server component failed", "error", runErr)
		cancel()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	slog.Info("shutting down")
	heartbeatService.Stop()
	cronService.Stop()
	chanMgr.StopAll(shutdownCtx)
	if err := gatewayServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("gateway shutdown failed", "error", err)
	}

	return runErr
}

func buildHeartbeatService(cfg *config.Config, msgBus *bus.MessageBus, cronService *cron.Service) *heartbeat.Service {
	return heartbeat.NewService(
		heartbeat.Config{
			Enabled:  cfg.Heartbeat.Enabled,
			Interval: time.Duration(cfg.Heartbeat.Interval) * time.Minute,
			MaxIdle:  time.Duration(cfg.Heartbeat.MaxIdleMinutes) * time.Minute,
		},
		func(ctx context.Context) (string, error) {
			status := cronService.Status()

			running, _ := status["running"].(bool)
			enabledJobs, _ := status["enabled_jobs"].(int)
			totalJobs, _ := status["total_jobs"].(int)
			return fmt.Sprintf("cron_running=%t enabled_jobs=%d total_jobs=%d", running, enabledJobs, totalJobs), nil
		},
		func(ctx context.Context, channel, chatID, content, requestID string) error {
			msgBus.PublishOutbound(&bus.OutboundMessage{
				Channel:   channel,
				ChatID:    chatID,
				Content:   content,
				RequestID: requestID,
				Metadata: map[string]any{
					"type": "heartbeat",
				},
			})
			return nil
		},
	)
}

func registerEnabledChannels(cfg *config.Config, msgBus *bus.MessageBus, chanMgr *channel.Manager) {
	register := func(ch channel.Channel) {
		chanMgr.Register(ch)
		slog.Info("channel registered", "name", ch.Name())
	}
	skip := func(name, reason string) {
		slog.Warn("channel enabled but not ready; skipping registration", "name", name, "reason", reason)
	}

	if cfg.Channels.Telegram.Enabled {
		if cfg.Channels.Telegram.Token == "" {
			skip("telegram", "token not set")
		} else {
			register(telegram.New(&cfg.Channels.Telegram, msgBus))
		}
	}

	if cfg.Channels.WhatsApp.Enabled {
		if cfg.Channels.WhatsApp.BridgeURL == "" {
			skip("whatsapp", "bridge_url not set")
		} else {
			register(whatsapp.New(&cfg.Channels.WhatsApp, msgBus))
		}
	}

	if cfg.Channels.Feishu.Enabled {
		if cfg.Channels.Feishu.AppID == "" || cfg.Channels.Feishu.AppSecret == "" {
			skip("feishu", "app_id/app_secret not set")
		} else {
			register(feishu.New(&cfg.Channels.Feishu, msgBus))
		}
	}

	if cfg.Channels.Discord.Enabled {
		if cfg.Channels.Discord.Token == "" {
			skip("discord", "token not set")
		} else {
			register(discord.New(&cfg.Channels.Discord, msgBus))
		}
	}

	if cfg.Channels.Slack.Enabled {
		if cfg.Channels.Slack.BotToken == "" || cfg.Channels.Slack.AppToken == "" {
			skip("slack", "bot_token/app_token not set")
		} else {
			register(slack.New(&cfg.Channels.Slack, msgBus))
		}
	}

	if cfg.Channels.QQ.Enabled {
		if cfg.Channels.QQ.AppID == "" || cfg.Channels.QQ.AppSecret == "" {
			skip("qq", "app_id/app_secret not set")
		} else {
			register(qq.New(&cfg.Channels.QQ, msgBus))
		}
	}

	if cfg.Channels.DingTalk.Enabled {
		if cfg.Channels.DingTalk.ClientID == "" || cfg.Channels.DingTalk.ClientSecret == "" {
			skip("dingtalk", "client_id/client_secret not set")
		} else {
			register(dingtalk.New(&cfg.Channels.DingTalk, msgBus))
		}
	}

	if cfg.Channels.MaixCam.Enabled {
		if cfg.Channels.MaixCam.Host == "" || cfg.Channels.MaixCam.Port <= 0 {
			skip("maixcam", "host/port not set")
		} else {
			register(maixcam.New(&cfg.Channels.MaixCam, msgBus))
		}
	}
}
