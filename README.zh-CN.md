# Golem (גּוֹלֶם)

<div align="center">

<img src="docs/logo.png" width="180" />

[![Go Version](https://img.shields.io/github/go-mod/go-version/MEKXH/golem?style=flat-square&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/MEKXH/golem?style=flat-square&logo=github)](https://github.com/MEKXH/golem/releases/latest)
[![CI Status](https://img.shields.io/github/actions/workflow/status/MEKXH/golem/ci.yml?style=flat-square&logo=github-actions)](https://github.com/MEKXH/golem/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/MEKXH/golem?style=flat-square)](LICENSE)

**你的 AI Agent，你的终端，你的规则。**

</div>

Golem 是一个以终端为中心的个人 AI 助手，基于 [Go](https://go.dev/) 和 [Eino](https://github.com/cloudwego/eino) 构建。
它支持对话、工具调用、Shell 命令执行、文件读写、网页搜索与抓取、长期记忆、Cron 定时任务、多渠道后台服务，以及 Provider 认证登录与渠道语音转写。

> **Golem (גולם)**：在犹太传说中，Golem 是由无生命物质塑造并被赋予行动能力的"仆从"。

## 文档导航

- [README (English)](README.md)
- [使用手册（英文）](docs/user-guide.md)
- [使用手册（简体中文）](docs/user-guide.zh-CN.md)
- [运维手册（英文）](docs/operations/runbook.md)
- [运维手册（简体中文）](docs/operations/runbook.zh-CN.md)

## 为什么选择 Golem

- 单一二进制，零运行时依赖膨胀（无需 Python/Node/Docker）。
- 模型提供商解耦，通过统一的 OpenAI 兼容层切换。
- 真正的 Agent 循环，支持工具调用，不是纯聊天壳子。
- 既可本地交互（`golem chat`），也可后台常驻（`golem run`）。
- 内置多渠道接入、Gateway API、Cron 调度、Heartbeat 探活和技能系统。
- 内置认证命令、语音转写链路和可跨重启恢复的心跳路由。

## 架构概览

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Golem 架构图                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────┐    │
│  │   渠道层     │     │   Agent      │     │     提供商       │    │
│  │ (Telegram,   │────▶│   循环       │────▶│ (Claude, OpenAI, │    │
│  │  Discord,    │     │              │     │  DeepSeek...)    │    │
│  │  Slack...)   │     └──────┬───────┘     └──────────────────┘    │
│  └──────────────┘            │                                       │
│         │                    │                                       │
│         ▼                    ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                         消息总线                             │    │
│  │              (入站/出站异步消息队列)                          │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                       │
│         ┌────────────────────┼────────────────────┐                 │
│         ▼                    ▼                    ▼                 │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────┐       │
│  │   会话      │     │   技能      │     │     工具        │       │
│  │  (历史)     │     │  (提示词)   │     │(exec, file, web)│       │
│  └─────────────┘     └─────────────┘     └─────────────────┘       │
│         │                    │                    │                 │
│         └────────────────────┼────────────────────┘                 │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                      支持服务层                              │    │
│  │    (记忆 | Cron | 心跳 | 网关 | 技能)                       │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 核心组件

| 组件 | 路径 | 描述 |
|------|------|------|
| **Agent 循环** | `internal/agent/` | 主处理循环，支持工具调用，默认最多 20 轮 |
| **消息总线** | `internal/bus/` | 基于 Go Channel 的事件驱动消息路由 |
| **渠道系统** | `internal/channel/` | 多平台集成（Telegram、Discord、Slack 等） |
| **提供商** | `internal/provider/` | 通过 Eino OpenAI 封装层统一 LLM 接口 |
| **会话** | `internal/session/` | 持久化 JSONL 格式对话历史 |
| **工具** | `internal/tools/` | 内置工具：文件、Shell、记忆、网页、Cron、消息、子 Agent |
| **记忆** | `internal/memory/` | 长期记忆与每日日记系统 |
| **技能** | `internal/skills/` | 可扩展的 Markdown 提示词包 |
| **Cron** | `internal/cron/` | 定时任务管理 |
| **心跳** | `internal/heartbeat/` | 定期健康探测与状态回传 |
| **网关** | `internal/gateway/` | HTTP API 服务器（`/health`、`/version`、`/chat`） |

## 核心能力

### 交互方式

- 终端 TUI 对话（`golem chat`）
- 多渠道机器人服务（`golem run`）：Telegram、WhatsApp、Feishu、Discord、Slack、QQ、DingTalk、MaixCam
- Gateway HTTP API（`/health`、`/version`、`/chat`）

### 最新能力

- 认证命令：`golem auth login`、`golem auth logout`、`golem auth status`
- Heartbeat 目标会话持久化：重启后自动恢复最近活跃的渠道/会话路由
- Telegram / Discord / Slack 音频消息自动转写，失败时回退占位文本，不阻断主流程
- 文件增量编辑工具：`edit_file` 与 `append_file`
- 策略守卫与审批流：`strict`/`relaxed`/`off`，支持 `off_ttl` 限时放开后自动回收
- MCP 动态工具接入：以 `mcp.<server>.<tool>` 注册并复用同一策略/审批链路

### 内置工具

| 工具 | 说明 |
| --- | --- |
| `exec` | 执行 Shell 命令（支持限制在工作区内） |
| `read_file` / `write_file` / `edit_file` / `append_file` | 在工作区中读取/写入/编辑/追加文件 |
| `list_dir` | 列出目录内容 |
| `read_memory` / `write_memory` | 读写长期记忆 |
| `append_diary` | 追加每日日志 |
| `web_search` | 网页搜索（有 Brave Key 优先使用 Brave） |
| `web_fetch` | 抓取并提取网页内容 |
| `manage_cron` | 管理定时任务 |
| `message` | 向渠道发送消息 |
| `spawn` / `subagent` | 委托任务给子 Agent |

### 支持的 LLM 提供商

OpenRouter、Claude、OpenAI、DeepSeek、Gemini、Ark、Qianfan、Qwen、Ollama。

### 子 Agent 系统

Golem 支持将任务委托给子 Agent 并行处理：

- **`spawn`**：异步子 Agent，立即返回任务 ID，通过消息总线通知结果
- **`subagent`**：同步子 Agent，阻塞直到完成，直接返回结果

两种模式都使用独立会话，并传播原始渠道/会话信息以便结果回传。

### 记忆系统

双层记忆架构：

1. **长期记忆**：单个 `MEMORY.md` 文件存储持久知识
2. **每日日记**：`YYYY-MM-DD.md` 文件存储带时间戳的日记条目

### Heartbeat 探活

在服务模式启用后，系统可以定期执行健康探测，并向最近活跃会话回传心跳结果。最近活跃目标会持久化到工作区状态文件中，服务重启后仍可恢复投递路由。

## 安装

### 方式 A：下载二进制（推荐）

从 [Releases](https://github.com/MEKXH/golem/releases) 下载 Windows/Linux 预编译文件。

### 方式 B：源码安装

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

## 快速开始

### 1. 初始化配置

```bash
golem init
```

该命令会生成 `~/.golem/config.json` 和工作区目录。

### 2. 使用示例配置模板

使用仓库内模板作为起点：

```bash
cp config/config.example.json ~/.golem/config.json
```

PowerShell:

```powershell
Copy-Item config/config.example.json "$HOME/.golem/config.json"
```

然后编辑 `~/.golem/config.json`，至少填入一个 provider 的 key（例如 `providers.openai.api_key`）。

建议同时基于模板创建环境变量文件（用于 local/staging/production 隔离）：

```bash
cp .env.example .env.local
```

PowerShell:

```powershell
Copy-Item .env.example .env.local
```

在 `.env.local` 中补齐必填密钥（至少一个 provider key；若网关对外暴露则必须设置 `GOLEM_GATEWAY_TOKEN`）。

可选（使用 token/OAuth 认证存储）：

```bash
golem auth login --provider openai --token "$OPENAI_API_KEY"
```

### 3. 运行 smoke 检查

```bash
make smoke
```

如果本机没有 `make`：

```bash
go test ./...
go run ./cmd/golem status
go run ./cmd/golem chat "ping"
```

### 4. 开始对话

```bash
golem chat
```

单次调用：

```bash
golem chat "分析当前目录结构"
```

### 5. 启动服务模式

```bash
golem run
```

## CLI 命令总览

| 命令 | 说明 |
| --- | --- |
| `golem init` | 初始化配置和工作区 |
| `golem chat [message]` | 启动 TUI 对话或单次发送消息 |
| `golem run` | 启动服务模式 |
| `golem status` | 查看系统状态摘要 |
| `golem auth login --provider <name> [--token <token> \| --device-code \| --browser]` | 通过 token 或 OAuth 保存 provider 凭据 |
| `golem auth logout [--provider <name>]` | 删除指定 provider 凭据，或删除全部凭据 |
| `golem auth status` | 查看当前认证凭据状态 |
| `golem channels list` | 列出已配置渠道 |
| `golem channels status` | 查看渠道详细状态 |
| `golem channels start <channel>` | 在配置中启用渠道 |
| `golem channels stop <channel>` | 在配置中停用渠道 |
| `golem cron list` | 列出定时任务 |
| `golem cron add -n <name> -m <msg> [--every <sec> \| --cron <expr> \| --at <ts>]` | 新增定时任务 |
| `golem cron run <job_id>` | 立即执行任务 |
| `golem cron remove <job_id>` | 删除任务 |
| `golem cron enable <job_id>` | 启用任务 |
| `golem cron disable <job_id>` | 禁用任务 |
| `golem approval list` | 列出待审批请求 |
| `golem approval approve <id> --by <name> [--note <text>]` | 通过审批请求 |
| `golem approval reject <id> --by <name> [--note <text>]` | 驳回审批请求 |
| `golem skills list` | 列出已安装技能 |
| `golem skills install <owner/repo>` | 从 GitHub 安装技能 |
| `golem skills remove <name>` | 删除技能 |
| `golem skills show <name>` | 查看技能内容 |
| `golem skills search [keyword]` | 搜索远程技能索引 |

## 认证说明

认证信息存储在 `~/.golem/auth.json`。当配置文件中的 provider key 为空时，Provider 会优先使用认证存储中的 token 作为调用凭据。

示例：

```bash
golem auth login --provider openai --device-code
golem auth status
golem auth logout --provider openai
```

## Cron 调度

支持三种任务类型：

- `--every <seconds>`：固定间隔执行
- `--cron "<expr>"`：标准 5 段 cron 表达式
- `--at "<RFC3339>"`：一次性执行

示例：

```bash
golem cron add -n "hourly-check" -m "检查系统状态并汇报" --every 3600
golem cron add -n "morning-brief" -m "给我一份晨间简报" --cron "0 9 * * *"
golem cron add -n "meeting-reminder" -m "提醒我参加团队会议" --at "2026-02-14T09:00:00Z"
```

## 技能系统

技能是会注入到 Agent 提示词中的 Markdown 指令包。

加载优先级：

1. `workspace/skills`
2. `~/.golem/skills`
3. 内置技能目录（默认 `~/.golem/builtin-skills`，可通过 `GOLEM_BUILTIN_SKILLS_DIR` 覆盖）

从 GitHub 安装：

```bash
golem skills install owner/repo
```

搜索远程技能：

```bash
golem skills search
golem skills search weather
```

## 配置说明

主配置文件：`~/.golem/config.json`
  
仓库模板文件：`config/config.example.json`

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
  "policy": {
    "mode": "strict",
    "off_ttl": "",
    "allow_persistent_off": false,
    "require_approval": ["exec"]
  },
  "mcp": {
    "servers": {
      "localfs": {
        "transport": "stdio",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "."]
      }
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
    },
    "voice": {
      "enabled": false,
      "provider": "openai",
      "model": "gpt-4o-mini-transcribe",
      "timeout_seconds": 30
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

`workspace_mode` 可选值：

- `default`：使用 `~/.golem/workspace`
- `cwd`：使用当前工作目录
- `path`：使用 `agents.defaults.workspace` 指定路径

`policy.mode` 可选值：

- `strict`：对 `require_approval` 中的工具强制走审批
- `relaxed`：允许执行，不触发审批
- `off`：关闭策略检查（建议搭配 `off_ttl` 仅限时放开）

审批与审计状态文件：

- `<workspace>/state/approvals.json`
- `<workspace>/state/audit.jsonl`

### 环境变量

所有配置项支持 `GOLEM_` 前缀：

```bash
export GOLEM_PROVIDERS_OPENROUTER_APIKEY="your-key"
export GOLEM_PROVIDERS_CLAUDE_APIKEY="your-key"
export GOLEM_LOG_LEVEL=debug
```

推荐按环境拆分文件：

- `.env.local`：本地开发
- `.env.staging`：预发布联调
- `.env.production`：生产部署

可从 `.env.example` 复制，并保持 `policy.mode=strict`、`policy.allow_persistent_off=false` 作为安全默认值。

最小必填密钥：

- 至少一个 provider API key（或使用 `golem auth login --provider <name>`）。
- 对外网络可访问的 staging/production 场景必须配置 `GOLEM_GATEWAY_TOKEN`。

## Gateway API

服务模式（`golem run`）下可用：

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

如果配置了 `gateway.token`，请求头需携带：

```text
Authorization: Bearer <token>
```

## 数据流

```
用户输入 (CLI/Telegram/Discord/Slack...)
         │
         ▼
    渠道层 (接收并验证消息)
         │
         ▼
    Bus.PublishInbound() ──▶ 消息总线.inbound
         │
         ▼
    Agent 循环 (处理消息)
         │
    ┌────┴────┐
    ▼         ▼         ▼
会话   上下文    LLM 生成
(历史) 构建器   (绑定工具)
              │           │
              │           ▼
              │      工具执行
              │      (工具调用)
              │           │
              └─────┬─────┘
                    ▼
         Bus.PublishOutbound()
                    │
                    ▼
         渠道管理器 (路由)
                    │
                    ▼
         渠道.Send() ──▶ 用户
```

## 引导文件

Agent 的系统提示词由以下文件构建（在工作区中搜索）：

1. `IDENTITY.md` - Agent 身份与人设
2. `SOUL.md` - 核心信念与价值观
3. `USER.md` - 用户特定上下文
4. `TOOLS.md` - 自定义工具描述
5. `AGENTS.md` - 子 Agent 定义

## 运维

故障处理、重启/回滚流程和生产建议请查看：

- [运维手册（英文）](docs/operations/runbook.md)
- [运维手册（简体中文）](docs/operations/runbook.zh-CN.md)

## 开发

提交前建议执行：

```bash
go test ./...
go test -race ./...
go vet ./...
```

构建：

```bash
go build -o golem ./cmd/golem
```

## 许可证

MIT
