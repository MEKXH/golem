# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Golem is a lightweight, extensible personal AI assistant built with Go and [Eino](https://github.com/cloudwego/eino). It provides both a Terminal UI (TUI) for interactive chat and a server mode for integrating with messaging platforms like Telegram.

## Development Commands

### Building
```bash
go build -o golem ./cmd/golem
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests in a specific package
go test ./internal/agent

# Run a specific test
go test -v -run TestFunctionName ./path/to/package
```

### Running Locally
```bash
# Initialize configuration
go run ./cmd/golem init

# Start interactive chat
go run ./cmd/golem chat

# Run as server
go run ./cmd/golem run

# Check server status
go run ./cmd/golem status
```

## Architecture

### Core Components

**Agent Loop** (`internal/agent/loop.go`)
- The main processing loop that orchestrates model interactions and tool execution
- Uses Eino's `ChatModel` interface for LLM communication
- Iteratively processes messages with tool calling support (max 20 iterations by default)
- Manages session history and workspace context

**Message Bus** (`internal/bus/`)
- Event-driven architecture for routing messages between channels and the agent
- `InboundMessage`: Messages received from channels (CLI, Telegram, etc.)
- `OutboundMessage`: Responses sent back to channels
- Uses Go channels for thread-safe message passing

**Channel System** (`internal/channel/`)
- Abstraction for different communication interfaces (CLI, Telegram)
- `Manager` coordinates multiple channels, routing messages to/from the message bus
- Each channel implements the `Channel` interface with `Start()`, `Send()`, and `Stop()`

**Session Management** (`internal/session/`)
- Persistent conversation history stored as JSONL files in `~/.golem/workspace/sessions/`
- Sessions keyed by `channel:chatID` (e.g., "telegram:123456")
- History limited to last 50 messages per session

**Provider System** (`internal/provider/`)
- Unified interface for multiple LLM providers via Eino's OpenAI-compatible wrapper
- Supports: OpenRouter, Claude, OpenAI, DeepSeek, Ollama, and others
- Model selection based on which provider has an API key configured

**Tool Registry** (`internal/tools/`)
- Thread-safe registry for managing agent tools
- Built-in tools:
  - `read_file`: Read files from workspace
  - `write_file`: Write files to workspace
  - `list_dir`: List directory contents
  - `exec`: Execute shell commands (optionally restricted to workspace)
- Tools follow Eino's `InvokableTool` interface

**Configuration** (`internal/config/`)
- JSON-based configuration at `~/.golem/config.json`
- Three workspace modes:
  - `default`: Uses `~/.golem/workspace`
  - `cwd`: Uses current working directory
  - `path`: Uses custom path specified in `agents.defaults.workspace`
- Environment variables supported with `GOLEM_` prefix

### Data Flow

1. User input arrives via a Channel (CLI TUI or Telegram)
2. Channel publishes `InboundMessage` to the message bus
3. Agent Loop receives message from bus's inbound queue
4. Loop builds context from session history and current message
5. Loop iteratively calls LLM with tool binding:
   - LLM responds with content and/or tool calls
   - Tools execute with results fed back to LLM
   - Continues until no tool calls or max iterations reached
6. Final response published as `OutboundMessage` to bus
7. Channel Manager routes outbound message to appropriate channel
8. Session history saved to disk

### Key Design Patterns

- **Event-Driven**: Message bus decouples channels from agent logic
- **Registry Pattern**: Tools and channels registered dynamically
- **OpenAI Compatibility**: All providers wrapped via OpenAI-compatible interface
- **Workspace Isolation**: All file operations scoped to configured workspace (unless disabled)

## Important Notes

- The agent uses Eino's tool calling mechanism with automatic schema binding
- Session files are JSONL (one JSON object per line) for append efficiency
- Tool execution is synchronous within the agent loop
- The TUI uses Bubble Tea framework (charmbracelet/bubbletea)
- Model names use provider prefix format: `anthropic/claude-sonnet-4-5`, `openai/gpt-4`, etc.
