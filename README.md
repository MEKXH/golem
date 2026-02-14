# Golem (גּוֹלֶם)

<div align="center">

<img src="docs/logo.png" width="180" />

[![Go Version](https://img.shields.io/github/go-mod/go-version/MEKXH/golem?style=flat-square&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/MEKXH/golem?style=flat-square&logo=github)](https://github.com/MEKXH/golem/releases/latest)
[![CI Status](https://img.shields.io/github/actions/workflow/status/MEKXH/golem/ci.yml?style=flat-square&logo=github-actions)](https://github.com/MEKXH/golem/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/MEKXH/golem?style=flat-square)](LICENSE)

**Your AI agent. Your terminal. Your rules.**

</div>

Golem is a terminal-first personal AI assistant built with [Go](https://go.dev/) and [Eino](https://github.com/cloudwego/eino).
It can chat, run tools, call shell commands, manage files, search/fetch web content, keep memory, schedule cron jobs, and run as a background service across multiple channels.

> **Golem (גולם)**: In Jewish folklore, a golem is an animated being made from inanimate matter, created to serve.

## Documentation

- [README (简体中文)](README.zh-CN.md)
- [Operations Runbook (English)](docs/operations/runbook.md)
- [运行手册（简体中文）](docs/operations/runbook.zh-CN.md)

## Why Golem

- One binary, zero runtime dependency bloat (no Python/Node/Docker required).
- Provider-agnostic model access through a unified OpenAI-compatible layer.
- Real agent loop with tool calling, not just plain text chat.
- Works both interactively (`golem chat`) and as long-running service (`golem run`).
- Built-in channels, gateway API, cron scheduler, heartbeat service, and skill system.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Golem Architecture                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────┐    │
│  │   Channels   │     │    Agent     │     │     Providers    │    │
│  │  (Telegram,  │────▶│    Loop      │────▶│  (Claude, OpenAI,│    │
│  │  Discord,    │     │              │     │   DeepSeek...)   │    │
│  │  Slack...)   │     └──────┬───────┘     └──────────────────┘    │
│  └──────────────┘            │                                       │
│         │                    │                                       │
│         ▼                    ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                         Message Bus                          │    │
│  │           (Inbound/Outbound async message queue)             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                       │
│         ┌────────────────────┼────────────────────┐                 │
│         ▼                    ▼                    ▼                 │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────┐       │
│  │   Session   │     │   Skills    │     │     Tools       │       │
│  │  (History)  │     │ (Prompts)   │     │(exec, file, web)│       │
│  └─────────────┘     └─────────────┘     └─────────────────┘       │
│         │                    │                    │                 │
│         └────────────────────┼────────────────────┘                 │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                     Supporting Services                      │    │
│  │    (Memory | Cron | Heartbeat | Gateway | Skills)           │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Core Components

| Component | Path | Description |
|-----------|------|-------------|
| **Agent Loop** | `internal/agent/` | Main processing loop with tool calling, max 20 iterations |
| **Message Bus** | `internal/bus/` | Event-driven message routing via Go channels |
| **Channel System** | `internal/channel/` | Multi-platform integrations (Telegram, Discord, Slack, etc.) |
| **Provider** | `internal/provider/` | Unified LLM interface via Eino's OpenAI wrapper |
| **Session** | `internal/session/` | Persistent JSONL-based conversation history |
| **Tools** | `internal/tools/` | Built-in tools: file, shell, memory, web, cron, message, subagent |
| **Memory** | `internal/memory/` | Long-term memory and daily diary system |
| **Skills** | `internal/skills/` | Extensible Markdown-based prompt packs |
| **Cron** | `internal/cron/` | Scheduled job management |
| **Heartbeat** | `internal/heartbeat/` | Periodic health probe and status reporting |
| **Gateway** | `internal/gateway/` | HTTP API server (`/health`, `/version`, `/chat`) |

## Core Features

### Interaction Modes

- Terminal TUI chat (`golem chat`)
- Multi-channel bot mode (`golem run`): Telegram, WhatsApp, Feishu, Discord, Slack, QQ, DingTalk, MaixCam
- Gateway HTTP API (`/health`, `/version`, `/chat`)

### Built-in Tools

| Tool | Description |
| --- | --- |
| `exec` | Run shell commands (workspace restriction supported) |
| `read_file` / `write_file` | Read/write files in workspace |
| `list_dir` | List directory contents |
| `read_memory` / `write_memory` | Persistent memory access |
| `append_diary` | Append daily notes |
| `web_search` | Web search (Brave when API key exists; fallback available) |
| `web_fetch` | Fetch and extract web page content |
| `manage_cron` | Manage scheduled jobs |
| `message` | Send messages to channels |
| `spawn` / `subagent` | Delegate tasks to subagents |

### LLM Providers

OpenRouter, Claude, OpenAI, DeepSeek, Gemini, Ark, Qianfan, Qwen, Ollama.

### Subagent System

Golem supports delegating tasks to subagents for parallel processing:

- **`spawn`**: Asynchronous subagent, returns task ID immediately, notifies via message bus
- **`subagent`**: Synchronous subagent, blocks until completion, returns result directly

Both modes use isolated sessions and propagate origin channel/chat for result delivery.

### Memory System

Two-tier memory architecture:

1. **Long-term Memory**: Single `MEMORY.md` file for persistent knowledge
2. **Daily Diary**: `YYYY-MM-DD.md` files for timestamped journal entries

### Heartbeat Service

When enabled, server mode can periodically run a health probe and send heartbeat output to the latest active channel/session.

## Installation

### Option A: Download Binary

Download Windows/Linux binaries from [Releases](https://github.com/MEKXH/golem/releases).

### Option B: Install from Source

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

## Quick Start

### 1. Initialize config

```bash
golem init
```

This creates `~/.golem/config.json`.

### 2. Configure provider credentials

Example:

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-sonnet-4-5",
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "claude": {
      "api_key": "your-api-key-here"
    }
  }
}
```

### 3. Start chatting

```bash
golem chat
```

One-shot:

```bash
golem chat "Analyze the current directory structure"
```

### 4. Start server mode

```bash
golem run
```

## CLI Commands

| Command | Description |
| --- | --- |
| `golem init` | Initialize config and workspace |
| `golem chat [message]` | Start TUI chat or send one-shot message |
| `golem run` | Start server mode |
| `golem status` | Show system status summary |
| `golem channels list` | List configured channels |
| `golem channels status` | Show detailed channel status |
| `golem channels start <channel>` | Enable one channel in config |
| `golem channels stop <channel>` | Disable one channel in config |
| `golem cron list` | List scheduled jobs |
| `golem cron add -n <name> -m <msg> [--every <sec> \| --cron <expr> \| --at <ts>]` | Add a job |
| `golem cron run <job_id>` | Run a job immediately |
| `golem cron remove <job_id>` | Remove a job |
| `golem cron enable <job_id>` | Enable a job |
| `golem cron disable <job_id>` | Disable a job |
| `golem skills list` | List installed skills |
| `golem skills install <owner/repo>` | Install skill from GitHub |
| `golem skills remove <name>` | Remove installed skill |
| `golem skills show <name>` | Show skill content |
| `golem skills search [keyword]` | Search remote skill index |

## Cron Scheduling

Schedule types:

- `--every <seconds>`: fixed interval
- `--cron "<expr>"`: standard 5-field cron expression
- `--at "<RFC3339>"`: one-shot execution

Examples:

```bash
golem cron add -n "hourly-check" -m "Check system status and report" --every 3600
golem cron add -n "morning-brief" -m "Give me a morning briefing" --cron "0 9 * * *"
golem cron add -n "meeting-reminder" -m "Remind me about the team meeting" --at "2026-02-14T09:00:00Z"
```

## Skills System

Skills are Markdown instruction packs loaded into the agent prompt.

Skill discovery precedence:

1. `workspace/skills`
2. `~/.golem/skills`
3. builtin skills directory (default: `~/.golem/builtin-skills`, override via `GOLEM_BUILTIN_SKILLS_DIR`)

Install from GitHub:

```bash
golem skills install owner/repo
```

Search remote skills:

```bash
golem skills search
golem skills search weather
```

## Configuration

Main file: `~/.golem/config.json`

```json
{
  "agents": {
    "defaults": {
      "workspace_mode": "default",
      "workspace": "",
      "model": "anthropic/claude-sonnet-4-5",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "channels": {
    "telegram": {
      "enabled": false,
      "token": "",
      "allow_from": []
    }
  },
  "providers": {
    "claude": {
      "api_key": ""
    },
    "openai": {
      "api_key": ""
    },
    "ollama": {
      "base_url": "http://localhost:11434"
    }
  },
  "tools": {
    "exec": {
      "timeout": 60,
      "restrict_to_workspace": true
    },
    "web": {
      "search": {
        "api_key": "",
        "max_results": 5
      }
    }
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790,
    "token": ""
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30,
    "max_idle_minutes": 720
  },
  "log": {
    "level": "info",
    "file": ""
  }
}
```

`workspace_mode` values:

- `default`: use `~/.golem/workspace`
- `cwd`: use current working directory
- `path`: use `agents.defaults.workspace`

### Environment Variables

All config keys support `GOLEM_` prefix:

```bash
export GOLEM_PROVIDERS_OPENROUTER_APIKEY="your-key"
export GOLEM_PROVIDERS_CLAUDE_APIKEY="your-key"
export GOLEM_LOG_LEVEL=debug
```

## Gateway API

Available in server mode (`golem run`):

- `GET /health`
- `GET /version`
- `POST /chat`

`POST /chat` example:

```json
{
  "message": "Summarize the latest logs",
  "session_id": "ops-room",
  "sender_id": "api-client"
}
```

If `gateway.token` is configured, include:

```text
Authorization: Bearer <token>
```

## Data Flow

```
User Input (CLI/Telegram/Discord/Slack...)
         │
         ▼
    Channel (receives & validates message)
         │
         ▼
    Bus.PublishInbound() ──▶ MessageBus.inbound
         │
         ▼
    Agent Loop (processes message)
         │
    ┌────┴────┐
    ▼         ▼         ▼
Session  Context  LLM Generate
(History) Builder  (with tools bound)
              │           │
              │           ▼
              │      Tools.Execute()
              │      (tool calls)
              │           │
              └─────┬─────┘
                    ▼
         Bus.PublishOutbound()
                    │
                    ▼
         Channel Manager (routes)
                    │
                    ▼
         Channel.Send() ──▶ User
```

## Bootstrap Files

The agent's system prompt is built from these files (searched in workspace):

1. `IDENTITY.md` - Agent identity and persona
2. `SOUL.md` - Core beliefs and values
3. `USER.md` - User-specific context
4. `TOOLS.md` - Custom tool descriptions
5. `AGENTS.md` - Subagent definitions

## Operations

For incident handling, restart/rollback flow, and production guidance:

- [Operations Runbook (English)](docs/operations/runbook.md)
- [运行手册（简体中文）](docs/operations/runbook.zh-CN.md)

## Development

Run before pushing:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Build:

```bash
go build -o golem ./cmd/golem
```

## License

MIT