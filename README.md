# Golem (גּוֹלֶם)

<div align="center">

[![Go Version](https://img.shields.io/github/go-mod/go-version/MEKXH/golem?style=flat-square&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/MEKXH/golem?style=flat-square&logo=github)](https://github.com/MEKXH/golem/releases/latest)
[![CI Status](https://img.shields.io/github/actions/workflow/status/MEKXH/golem/release.yml?style=flat-square&logo=github-actions)](https://github.com/MEKXH/golem/actions)
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
  - **Web Search**: Integrated web search capabilities.
- **🔌 Multi-Provider Support**: Seamlessly switch between OpenAI, Claude, DeepSeek, Ollama, Gemini, and more.

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
      "model": "anthropic/claude-3-5-sonnet-20241022"
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

## Configuration

The configuration file is located at `~/.golem/config.json`. Below is a comprehensive example:

```json
{
  "agents": {
    "defaults": {
      "workspace_mode": "default", // Options: "default" (~/.golem/workspace), "cwd", "path"
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
      "restrict_to_workspace": false
    },
    "web": {
      "search": {
        "api_key": "YOUR_BRAVE_SEARCH_API_KEY", // Optional
        "max_results": 5
      }
    }
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
```

## License

MIT
