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
	"github.com/MEKXH/golem/internal/channel/telegram"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
	"github.com/MEKXH/golem/internal/gateway"
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

	errCh := make(chan error, 2)
	go func() {
		if err := loop.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("agent loop failed: %w", err)
		}
	}()

	chanMgr := channel.NewManager(msgBus)

	if cfg.Channels.Telegram.Enabled {
		tg := telegram.New(&cfg.Channels.Telegram, msgBus)
		chanMgr.Register(tg)
	}

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
	cronService.Stop()
	chanMgr.StopAll(shutdownCtx)
	if err := gatewayServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("gateway shutdown failed", "error", err)
	}

	return runErr
}
