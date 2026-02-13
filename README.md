# Golem (גּוֹלֶם)

<div align="center">

[![Go Version](https://img.shields.io/github/go-mod/go-version/MEKXH/golem?style=flat-square&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/MEKXH/golem?style=flat-square&logo=github)](https://github.com/MEKXH/golem/releases/latest)
[![CI Status](https://img.shields.io/github/actions/workflow/status/MEKXH/golem/ci.yml?style=flat-square&logo=github-actions)](https://github.com/MEKXH/golem/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/MEKXH/golem?style=flat-square)](LICENSE)

**Your AI agent. Your terminal. Your rules.**

</div>

Golem is a personal AI assistant that lives in your terminal and works while you sleep. Built with [Go](https://go.dev/) and [Eino](https://github.com/cloudwego/eino), it connects to any major LLM provider, executes shell commands, manages files, searches the web, and runs scheduled tasks — all from a single binary with zero external dependencies.

> **Golem (גּוֹלֶם)**: In Jewish folklore, a Golem is an animated being created from inanimate matter — an obedient servant that tirelessly performs tasks for its creator.

[中文文档](README.zh-CN.md)

---

## Why Golem?

- **One binary, zero bloat.** No Python, no Node, no Docker. Just a single Go binary.
- **Provider-agnostic.** Switch between 9 LLM providers with a config change — OpenRouter, Claude, OpenAI, DeepSeek, Gemini, Ark, Qianfan, Qwen, or Ollama.
- **Always-on option.** Run as a background server with Telegram integration, HTTP gateway, and cron-scheduled tasks.
- **Tool-wielding agent.** Not just a chatbot — Golem reads files, runs commands, searches the web, and remembers context across sessions.
- **Extensible by design.** Install skill packs from GitHub to teach it new tricks.

---

## Architecture

```
┌──────────────────────────────────────────────────┐
│              Channels (Input/Output)             │
│         CLI TUI  ·  Telegram  ·  Gateway API     │
└────────────────────────┬─────────────────────────┘
                         │
              ┌──────────▼──────────┐
              │     Message Bus     │
              │  (event-driven, Go  │
              │      channels)      │
              └──────────┬──────────┘
                         │
              ┌──────────▼──────────┐
              │     Agent Loop      │
              │ (iterative LLM +    │
              │   tool calling)     │
              └──────────┬──────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐    ┌──────▼──────┐   ┌────▼────┐
   │   LLM   │    │    Tools    │   │ Storage │
   │Providers│    │             │   │         │
   └─────────┘    └─────────────┘   └─────────┘
   OpenRouter      exec              Sessions
   Claude          read_file         (JSONL)
   OpenAI          write_file        Memory
   DeepSeek        list_dir          Skills
   Gemini          read_memory       Cron Jobs
   Ark             write_memory
   Qianfan         append_diary
   Qwen            web_search
   Ollama          web_fetch
                   manage_cron
```

---

## ✨ Features

### 🖥️ Interactive Terminal UI
A rich chat experience powered by Bubble Tea — autocomplete, streaming responses, and full tool execution, all in your terminal.

### 🤖 Server Mode
Run `golem run` as a background service. Connects Telegram, the HTTP Gateway API, and the cron scheduler simultaneously.

### 🛠️ 10 Built-in Tools

| Tool | What it does |
|------|-------------|
| `exec` | Run shell commands (sandboxing available) |
| `read_file` / `write_file` | Read and write files in the workspace |
| `list_dir` | Browse directory contents |
| `read_memory` / `write_memory` | Persistent long-term memory |
| `append_diary` | Daily diary notes |
| `web_search` | Search the web (Brave when API key exists, otherwise free DuckDuckGo fallback) |
| `web_fetch` | Fetch and extract web page content |
| `manage_cron` | Create and manage scheduled tasks |

### 🔌 9 LLM Providers
OpenRouter · Claude (Anthropic) · OpenAI · DeepSeek · Gemini · Ark · Qianfan · Qwen · Ollama — all via a unified OpenAI-compatible interface.

### ⏰ Cron Scheduler
Persistent, auto-resuming scheduled tasks. Supports one-shot (`at`), interval (`every`), and cron expressions. The agent can even schedule its own tasks.

### 🧩 Skills System
Install markdown-based skill packs from GitHub to extend the agent's system prompt with domain-specific instructions.

### 📡 Gateway API
HTTP endpoints (`/health`, `/version`, `/chat`) available during `golem run` for programmatic access from external services.

---

## Installation

### Download Binary (Recommended)

Grab the pre-compiled binary for Windows or Linux from the [Releases](https://github.com/MEKXH/golem/releases) page.

### Install from Source

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

---

## Quick Start

**1. Initialize** — generates `~/.golem/config.json`:

```bash
golem init
```

**2. Add your API key** — edit `~/.golem/config.json`:

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

**3. Chat:**

```bash
golem chat
```

Or one-shot:

```bash
golem chat "Analyze the current directory structure"
```

**4. Go server mode** (Telegram + Gateway + Cron):

```bash
golem run
```

---

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

---

## Cron Scheduling

Jobs persist across restarts and can be managed via the CLI or by the agent itself using the `manage_cron` tool.

### Schedule Types

- **`--every <seconds>`** — Repeat at a fixed interval (e.g., `--every 3600` for hourly).
- **`--cron <expr>`** — Standard 5-field cron expression (e.g., `--cron "0 9 * * *"` for daily at 9 AM).
- **`--at <timestamp>`** — One-shot at an RFC3339 timestamp (auto-deleted after run).

### Examples

```bash
# Hourly system check
golem cron add -n "hourly-check" -m "Check system status and report" --every 3600

# Daily morning briefing
golem cron add -n "morning-brief" -m "Give me a morning briefing" --cron "0 9 * * *"

# One-time reminder
golem cron add -n "meeting" -m "Remind me about the team meeting" --at "2026-02-14T09:00:00Z"
```

---

## Skills System

Skills are markdown instruction packs loaded into the agent's system prompt automatically.

### Skill File Format

Each skill lives in `workspace/skills/<name>/` with a `SKILL.md` file:

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

Downloads the `SKILL.md` from the repository's main branch.

---

## Configuration

Located at `~/.golem/config.json`:

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
        "api_key": "YOUR_BRAVE_SEARCH_API_KEY (optional)",
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

---

## Gateway API

Available when `golem run` is active.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/version` | GET | Version info |
| `/chat` | POST | Send a message to the agent |

**`POST /chat` example:**

```json
{
  "message": "Summarize the latest logs",
  "session_id": "ops-room",
  "sender_id": "api-client"
}
```

If `gateway.token` is set, include `Authorization: Bearer <token>`.

---

## Development

### Local Quality Checks

Run before pushing:

```bash
go test ./...
go test -race ./...
go vet ./...
```

### Branch and PR Workflow

1. Create a focused feature branch: `feature/<phase>-<topic>`
2. Keep PRs small and aligned with one phase/task
3. Merge to `main` only after CI is green

---

## License

MIT
