package commands

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/MEKXH/golem/internal/agent"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/config"
    "github.com/MEKXH/golem/internal/provider"
    "github.com/spf13/cobra"
)

func NewChatCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "chat [message]",
        Short: "Chat with Golem",
        RunE:  runChat,
    }
}

func runChat(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    model, err := provider.NewChatModel(ctx, cfg)
    if err != nil {
        fmt.Printf("Warning: %v\n", err)
        fmt.Println("Running without LLM (tools only mode)")
        model = nil
    }

    msgBus := bus.NewMessageBus(10)
    loop := agent.NewLoop(cfg, msgBus, model)

    if err := loop.RegisterDefaultTools(cfg); err != nil {
        return fmt.Errorf("failed to register tools: %w", err)
    }

    if len(args) > 0 {
        message := strings.Join(args, " ")
        resp, err := loop.ProcessDirect(ctx, message)
        if err != nil {
            return err
        }
        fmt.Println(resp)
        return nil
    }

    fmt.Println("Golem ready. Type 'exit' to quit.")
    scanner := bufio.NewScanner(os.Stdin)

    for {
        fmt.Print("\n> ")
        if !scanner.Scan() {
            break
        }

        input := strings.TrimSpace(scanner.Text())
        if input == "exit" || input == "quit" {
            break
        }
        if input == "" {
            continue
        }

        resp, err := loop.ProcessDirect(ctx, input)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        fmt.Println(resp)
    }

    return nil
}
