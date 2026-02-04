//go:build tools
// +build tools

package tools

import (
    _ "github.com/cloudwego/eino"
    _ "github.com/cloudwego/eino-ext/components/model/claude"
    _ "github.com/cloudwego/eino-ext/components/model/ollama"
    _ "github.com/cloudwego/eino-ext/components/model/openai"
    _ "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    _ "github.com/spf13/cobra"
    _ "github.com/spf13/viper"
)
