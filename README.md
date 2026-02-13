# Golem (גּוֹלֶם)

<div align="center">

[![Go Version](https://img.shields.io/github/go-mod/go-version/MEKXH/golem?style=flat-square&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/MEKXH/golem?style=flat-square&logo=github)](https://github.com/MEKXH/golem/releases/latest)
[![CI Status](https://img.shields.io/github/actions/workflow/status/MEKXH/golem/ci.yml?style=flat-square&logo=github-actions)](https://github.com/MEKXH/golem/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/MEKXH/golem?style=flat-square)](LICENSE)

_A modern, extensible AI assistant for your terminal and beyond._

</div>

**Golem** is a lightweight, extensible personal AI assistant built with [Go](https://go.dev/) and [Eino](https://github.com/cloudwego/eino). It allows you to run a powerful AI agent locally effectively using your terminal or through messaging platforms like Telegram.

> **Golem (גּוֹלֶם)**: In Jewish folklore, a Golem is an animated anthropomorphic being that is magically created entirely from inanimate matter (specifically clay or mud). It is an obedient servant that performs tasks for its creator.

[中文文档](README.zh-CN.md)

## ✨ Features

- **🖥️ Terminal User Interface (TUI)**: A rich, interactive chat experience comfortably within your terminal.
- **🤖 Server Mode**: Run Golem as a background service to interact via external channels (currently supports **Telegram**).
- **🛠️ Tool Use**:
  - **Shell Execution**: The agent can run system commands (safe mode available).
  - **File System**: Read and manipulate files within a designated workspace.
  - **Memory Tools**: Read/write long-term memory and append daily diary notes.
  - **Web Search & Fetch**: Search with Brave API (when configured) and fetch web page content.
  - **Cron Jobs**: Create, manage, and schedule recurring tasks that the agent executes automatically.
- **🔌 Multi-Provider Support**: Seamlessly switch between OpenAI, Claude, DeepSeek, Ollama, Gemini, and more.
- **⏰ Cron Scheduling System**: Built-in scheduler supports one-shot (`at`), interval (`every`), and cron expression schedules with persistent storage.
- **🧩 Skills System**: Install, manage, and load skill packs from GitHub to extend the agent's capabilities.
- **📡 Channel Management**: Inspect and manage communication channels from the CLI.
- **Workspace Management**: Sandboxed execution environments for safety and context management.

## Installation

### Download Binary (Recommended)

You can download the pre-compiled binary for Windows or Linux from the [Releases](https://github.com/MEKXH/golem/releases) page.

### Install from Source

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

## Quick Start

### 1. Initialize Configuration

Generate the default configuration file at `~/.golem/config.json`:

```bash
golem init
```

### 2. Configure Your Provider

Edit `~/.golem/config.json` to add your API keys. For example, to use Anthropic's Claude:

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-4-5-sonnet-20250929"
    }
  },
  "providers": {
    "claude": {
      "api_key": "your-api-key-here"
    }
  }
}
```

### 3. Start Chatting

Launch the interactive TUI:

```bash
golem chat
```

Or send a one-off message:

```bash
golem chat "Analyze the current directory structure"
```

### 4. Run as Server (Telegram Bot)

To use Golem via Telegram:

1.  Set `channels.telegram.enabled` to `true` in `config.json`.
2.  Add your Bot Token and allowed User IDs.
3.  Start the server:

```bash
golem run
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `golem init` | Initialize configuration and workspace |
| `golem chat` | Start interactive TUI chat |
| `golem run` | Start server mode (Telegram + Gateway + Cron) |
| `golem status` | Show system status (providers, channels, cron, skills) |
| `golem channels list` | List all configured channels |
| `golem channels status` | Show detailed channel status |
| `golem cron list` | List all scheduled jobs |
| `golem cron add -n <name> -m <msg> [--every <sec> \| --cron <expr> \| --at <ts>]` | Add a scheduled job |
| `golem cron remove <id>` | Remove a scheduled job |
| `golem cron enable <id>` | Enable a scheduled job |
| `golem cron disable <id>` | Disable a scheduled job |
| `golem skills list` | List installed skills |
| `golem skills install <repo>` | Install a skill from GitHub |
| `golem skills remove <name>` | Remove an installed skill |
| `golem skills show <name>` | Show skill content |

## Cron Scheduling

Golem includes a built-in cron scheduling system. Jobs persist across restarts and can be managed via the CLI or by the agent itself using the `manage_cron` tool.

### Schedule Types

- **`--every <seconds>`**: Repeat at a fixed interval (e.g., `--every 3600` for every hour).
- **`--cron <expr>`**: Standard 5-field cron expression (e.g., `--cron "0 9 * * *"` for daily at 9 AM).
- **`--at <timestamp>`**: One-shot execution at an RFC3339 timestamp (auto-deleted after run).

### Examples

```bash
# Remind every hour
golem cron add -n "hourly-check" -m "Check system status and report" --every 3600

# Daily morning briefing
golem cron add -n "morning-brief" -m "Give me a morning briefing" --cron "0 9 * * *"

# One-time reminder
golem cron add -n "meeting" -m "Remind me about the team meeting" --at "2026-02-14T09:00:00Z"
```

## Skills System

Skills are markdown-based instruction packs that extend the agent's capabilities. They are loaded into the system prompt automatically.

### Skill File Format

Each skill is a directory under `workspace/skills/<name>/` containing a `SKILL.md` file:

```markdown
---
name: weather
description: "Query weather information"
---

# Weather Skill
(Skill instructions for the agent)
```

### Install from GitHub

```bash
golem skills install owner/repo
```

This downloads the `SKILL.md` from the repository's main branch.

## Configuration

The configuration file is located at `~/.golem/config.json`. Below is a comprehensive example:

```json
{
  "agents": {
    "defaults": {
      "workspace_mode": "default",
      "model": "anthropic/claude-4-5-sonnet-20250929",
      "max_tokens": 8192,
      "temperature": 0.7
    }
  },
  "channels": {
    "telegram": {
      "enabled": false,
      "token": "YOUR_TELEGRAM_BOT_TOKEN",
      "allow_from": ["YOUR_TELEGRAM_USER_ID"]
    }
  },
  "providers": {
    "openai": { "api_key": "sk-..." },
    "claude": { "api_key": "sk-ant-..." },
    "ollama": { "base_url": "http://localhost:11434" }
  },
  "tools": {
    "exec": {
      "timeout": 60,
      "restrict_to_workspace": true
    },
    "web": {
      "search": {
        "api_key": "YOUR_BRAVE_SEARCH_API_KEY",
        "max_results": 5
      }
    }
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790,
    "token": "YOUR_GATEWAY_BEARER_TOKEN"
  },
  "log": {
    "level": "info",
    "file": ""
  }
}
```

## Gateway API

When `golem run` is started, Gateway HTTP endpoints are available:

- `GET /health`
- `GET /version`
- `POST /chat`

`POST /chat` request example:

```json
{
  "message": "Summarize the latest logs",
  "session_id": "ops-room",
  "sender_id": "api-client"
}
```

If `gateway.token` is set, send `Authorization: Bearer <token>`.

## Development

### Local Quality Checks

Run these commands before pushing:

```bash
go test ./...
go test -race ./...
go vet ./...
```

If any command fails, fix the issue and rerun all checks.

### Branch and PR Workflow

1. Create a focused feature branch: `feature/<phase>-<topic>`
2. Keep the PR scope small and aligned with one phase/task
3. Open a PR to `main` and merge only after CI is green

## License

MIT
