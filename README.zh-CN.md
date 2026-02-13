# Golem (גּוֹלֶם)

<div align="center">

[![Go Version](https://img.shields.io/github/go-mod/go-version/MEKXH/golem?style=flat-square&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/MEKXH/golem?style=flat-square&logo=github)](https://github.com/MEKXH/golem/releases/latest)
[![CI Status](https://img.shields.io/github/actions/workflow/status/MEKXH/golem/ci.yml?style=flat-square&logo=github-actions)](https://github.com/MEKXH/golem/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/MEKXH/golem?style=flat-square)](LICENSE)

_一个现代化的、可扩展的终端 AI 助手。_

</div>

**Golem** 是一个基于 [Go](https://go.dev/) 和 [Eino](https://github.com/cloudwego/eino) 构建的轻量级、可扩展的个人 AI 助手。它允许你通过终端或 Telegram 等消息平台，在本地高效运行强大的 AI 智能体。

> **Golem (גּוֹלֶם)**: 在犹太传说中，Golem（戈里姆/泥人）是一种被赋予生命的假人，通常由泥土或粘土制成。它是一个忠诚的仆人，不知疲倦地为创造者执行任务。

[English Documentation](README.md)

## ✨ 功能特性

- **🖥️ 终端用户界面 (TUI)**: 在终端内提供丰富、交互流畅的聊天体验。
- **🤖 服务端模式**: 将 Golem 作为后台服务运行，支持通过外部渠道交互（目前支持 **Telegram**）。
- **🛠️ 工具调用能力**:
  - **Shell 执行**: 智能体可以执行系统命令（提供安全模式）。
  - **文件系统**: 在指定工作区内读取和操作文件。
  - **记忆工具**: 支持读取/写入长期记忆并追加每日日记。
  - **网络搜索与抓取**: 可配置 Brave 搜索，并支持网页内容抓取。
  - **定时任务**: 创建、管理和调度由智能体自动执行的周期性任务。
- **🔌 多模型支持**: 无缝切换 OpenAI, Claude, DeepSeek, Ollama, Gemini 等多种模型提供商。
- **⏰ Cron 调度系统**: 内置调度器，支持一次性（`at`）、固定间隔（`every`）和 cron 表达式三种调度模式，任务持久化存储。
- **🧩 技能系统**: 从 GitHub 安装、管理和加载技能包，扩展智能体的能力。
- **📡 渠道管理**: 通过 CLI 检查和管理通信渠道。
- **工作区管理**: 提供沙箱化的执行环境，确保安全和上下文隔离。

## 安装指南

### 下载二进制文件 (推荐)

你可以从 [Releases](https://github.com/MEKXH/golem/releases) 页面下载适用于 Windows 或 Linux 的预编译二进制文件。

### 源码安装

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

## 快速开始

### 1. 初始化配置

在 `~/.golem/config.json` 生成默认配置文件：

```bash
golem init
```

### 2. 配置模型提供商

编辑 `~/.golem/config.json` 添加你的 API Key。例如使用 Anthropic Claude：

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

### 3. 开始对话

启动交互式 TUI：

```bash
golem chat
```

或者发送单条消息：

```bash
golem chat "分析当前目录结构"
```

### 4. 运行服务端 (Telegram Bot)

要通过 Telegram 使用 Golem：

1.  在 `config.json` 中设置 `channels.telegram.enabled` 为 `true`。
2.  填写你的 Bot Token 和允许的用户 ID (`allow_from`)。
3.  启动服务：

```bash
golem run
```

## CLI 命令一览

| 命令 | 说明 |
|------|------|
| `golem init` | 初始化配置和工作区 |
| `golem chat` | 启动交互式 TUI 聊天 |
| `golem run` | 启动服务端模式（Telegram + Gateway + Cron） |
| `golem status` | 显示系统状态（提供商、渠道、定时任务、技能） |
| `golem channels list` | 列出所有已配置渠道 |
| `golem channels status` | 显示渠道详细状态 |
| `golem cron list` | 列出所有定时任务 |
| `golem cron add -n <名称> -m <消息> [--every <秒> \| --cron <表达式> \| --at <时间戳>]` | 添加定时任务 |
| `golem cron remove <id>` | 删除定时任务 |
| `golem cron enable <id>` | 启用定时任务 |
| `golem cron disable <id>` | 禁用定时任务 |
| `golem skills list` | 列出已安装技能 |
| `golem skills install <repo>` | 从 GitHub 安装技能 |
| `golem skills remove <名称>` | 移除已安装技能 |
| `golem skills show <名称>` | 查看技能内容 |

## Cron 定时调度

Golem 内置了定时调度系统。任务跨重启持久化，可通过 CLI 或智能体自身的 `manage_cron` 工具进行管理。

### 调度类型

- **`--every <秒>`**: 固定间隔重复执行（如 `--every 3600` 表示每小时执行）。
- **`--cron <表达式>`**: 标准 5 字段 cron 表达式（如 `--cron "0 9 * * *"` 表示每天早上 9 点）。
- **`--at <时间戳>`**: 一次性执行，接受 RFC3339 格式时间戳（执行后自动删除）。

### 示例

```bash
# 每小时检查一次
golem cron add -n "hourly-check" -m "检查系统状态并汇报" --every 3600

# 每日早间简报
golem cron add -n "morning-brief" -m "给我一份早间简报" --cron "0 9 * * *"

# 一次性提醒
golem cron add -n "meeting" -m "提醒我参加团队会议" --at "2026-02-14T09:00:00Z"
```

## 技能系统

技能是基于 Markdown 的指令包，用于扩展智能体的能力。它们会被自动加载到系统提示中。

### 技能文件格式

每个技能是 `workspace/skills/<名称>/` 目录下的一个 `SKILL.md` 文件：

```markdown
---
name: weather
description: "查询天气信息"
---

# Weather Skill
（技能指令内容，智能体会据此执行相关任务）
```

### 从 GitHub 安装

```bash
golem skills install owner/repo
```

此命令会从仓库的 main 分支下载 `SKILL.md` 文件。

## 配置说明

配置文件位于 `~/.golem/config.json`。以下是一个完整的配置示例：

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

### workspace_mode 说明

| 模式 | 说明 |
|------|------|
| `default` | 使用 `~/.golem/workspace`（默认） |
| `cwd` | 使用当前工作目录 |
| `path` | 使用 `agents.defaults.workspace` 指定的自定义路径 |

## Gateway API

执行 `golem run` 后，可通过 HTTP 访问以下端点：

- `GET /health`
- `GET /version`
- `POST /chat`

`POST /chat` 请求示例：

```json
{
  "message": "总结最新日志",
  "session_id": "ops-room",
  "sender_id": "api-client"
}
```

如果配置了 `gateway.token`，请在请求头中携带 `Authorization: Bearer <token>`。

## 开发规范

### 本地质量检查

在推送代码前，请先执行：

```bash
go test ./...
go test -race ./...
go vet ./...
```

如果任一命令失败，请修复后重新执行全部检查。

### 分支与 PR 流程

1. 创建聚焦的功能分支：`feature/<phase>-<topic>`
2. 单个 PR 保持小范围，并与一个阶段/任务对齐
3. 向 `main` 发起 PR，且仅在 CI 全绿后合并

## 许可证

MIT
