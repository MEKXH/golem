package commands

import (
    "context"
    "fmt"
    "log/slog"
    "os/signal"
    "syscall"

    "github.com/MEKXH/golem/internal/agent"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/channel"
    "github.com/MEKXH/golem/internal/channel/telegram"
    "github.com/MEKXH/golem/internal/config"
    "github.com/MEKXH/golem/internal/provider"
    "github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
    var port int

    cmd := &cobra.Command{
        Use:   "run",
        Short: "Start Golem server",
        RunE:  runServer,
    }

    cmd.Flags().IntVarP(&port, "port", "p", 18790, "Server port")
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
    go loop.Run(ctx)

    chanMgr := channel.NewManager(msgBus)

    if cfg.Channels.Telegram.Enabled {
        tg := telegram.New(&cfg.Channels.Telegram, msgBus)
        chanMgr.Register(tg)
    }

    chanMgr.StartAll(ctx)
    go chanMgr.RouteOutbound(ctx)

    fmt.Printf("Golem server running. Press Ctrl+C to stop.\n")

    <-ctx.Done()

    slog.Info("shutting down")
    chanMgr.StopAll(context.Background())

    return nil
}
