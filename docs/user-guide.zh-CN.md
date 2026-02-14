# Golem 使用手册（简体中文）

English version: [docs/user-guide.md](user-guide.md)

本文档是 Golem 的完整使用与运维手册，覆盖当前版本的全部配置项、功能能力和操作流程。

## 1. Golem 是什么

Golem 是一个终端优先的个人 AI 助手，基于 Go + Eino 构建，支持：

- 交互式对话（`golem chat`）
- 常驻多渠道服务（`golem run`）
- 可调用工具的 Agent 循环（文件、Shell、记忆、网页、Cron、消息、子 Agent）
- 技能系统（工作区/全局/内置）
- Gateway HTTP API
- 认证存储（Token/OAuth 登录）
- Telegram/Discord/Slack 音频转写
- 心跳探活并回传到最近活跃会话

## 2. 安装与升级

## 2.1 源码安装

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

## 2.2 本地构建

```bash
go build -o golem ./cmd/golem
```

## 2.3 校验安装

```bash
golem --help
golem status
```

## 3. 首次初始化与目录结构

## 3.1 初始化

```bash
golem init
```

该命令会创建基础配置、工作区目录和内置技能。

## 3.2 关键路径说明

| 路径 | 作用 |
| --- | --- |
| `~/.golem/config.json` | 主配置文件 |
| `~/.golem/auth.json` | Provider 认证凭据存储 |
| `~/.golem/builtin-skills/` | `golem init` 写入的内置技能 |
| `<workspace>/memory/MEMORY.md` | 长期记忆 |
| `<workspace>/memory/YYYY-MM-DD.md` | 每日日记 |
| `<workspace>/skills/` | 工作区技能目录 |
| `<workspace>/sessions/*.jsonl` | 会话历史持久化 |
| `<workspace>/cron/jobs.json` | Cron 任务持久化 |
| `<workspace>/state/heartbeat.json` | 心跳目标会话持久化 |

`<workspace>` 由 `agents.defaults.workspace_mode` 决定：

- `default`：`~/.golem/workspace`
- `cwd`：当前工作目录
- `path`：使用 `agents.defaults.workspace`

## 4. 最小可用启动流程

```bash
golem init
cp config/config.example.json ~/.golem/config.json
# 编辑 key，例如 providers.openai.api_key
golem status
golem chat "ping"
```

服务模式：

```bash
golem run
```

## 5. 完整配置说明

主配置文件：`~/.golem/config.json`

## 5.1 全量配置示例

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
    "telegram": { "enabled": false, "token": "", "allow_from": [] },
    "whatsapp": { "enabled": false, "bridge_url": "", "allow_from": [] },
    "feishu": {
      "enabled": false,
      "app_id": "",
      "app_secret": "",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "discord": { "enabled": false, "token": "", "allow_from": [] },
    "slack": { "enabled": false, "bot_token": "", "app_token": "", "allow_from": [] },
    "qq": { "enabled": false, "app_id": "", "app_secret": "", "allow_from": [] },
    "dingtalk": { "enabled": false, "client_id": "", "client_secret": "", "allow_from": [] },
    "maixcam": { "enabled": false, "host": "0.0.0.0", "port": 9000, "allow_from": [] }
  },
  "providers": {
    "openrouter": { "api_key": "", "secret_key": "", "base_url": "" },
    "claude": { "api_key": "", "secret_key": "", "base_url": "" },
    "openai": { "api_key": "", "secret_key": "", "base_url": "" },
    "deepseek": { "api_key": "", "secret_key": "", "base_url": "" },
    "gemini": { "api_key": "", "secret_key": "", "base_url": "" },
    "ark": { "api_key": "", "secret_key": "", "base_url": "" },
    "qianfan": { "api_key": "", "secret_key": "", "base_url": "" },
    "qwen": { "api_key": "", "secret_key": "", "base_url": "" },
    "ollama": { "api_key": "", "secret_key": "", "base_url": "http://localhost:11434" }
  },
  "tools": {
    "exec": { "timeout": 60, "restrict_to_workspace": true },
    "web": { "search": { "api_key": "", "max_results": 5 } },
    "voice": {
      "enabled": false,
      "provider": "openai",
      "model": "gpt-4o-mini-transcribe",
      "timeout_seconds": 30
    }
  },
  "gateway": { "host": "0.0.0.0", "port": 18790, "token": "" },
  "heartbeat": { "enabled": true, "interval": 30, "max_idle_minutes": 720 },
  "log": { "level": "info", "file": "" }
}
```

## 5.2 `agents.defaults`

| 键 | 类型 | 默认值 | 约束 |
| --- | --- | --- | --- |
| `workspace_mode` | string | `default` | 只能是 `default`/`cwd`/`path` |
| `workspace` | string | `~/.golem/workspace` | 当 mode=`path` 时必填 |
| `model` | string | `anthropic/claude-sonnet-4-5` | 前缀影响 provider 选择 |
| `max_tokens` | int | `8192` | 必须 `> 0` |
| `temperature` | float | `0.7` | 范围 `[0, 2.0]` |
| `max_tool_iterations` | int | `20` | 非负；`0` 会回填为 `20` |

## 5.3 `channels.*`

| 键 | 类型 | 默认值 | 启用后是否必填 |
| --- | --- | --- | --- |
| `channels.telegram.enabled` | bool | `false` | - |
| `channels.telegram.token` | string | `""` | 是 |
| `channels.telegram.allow_from` | array | `[]` | 可选发送者白名单 |
| `channels.whatsapp.bridge_url` | string | `""` | 是 |
| `channels.feishu.app_id` | string | `""` | 是 |
| `channels.feishu.app_secret` | string | `""` | 是 |
| `channels.feishu.encrypt_key` | string | `""` | 否 |
| `channels.feishu.verification_token` | string | `""` | 否 |
| `channels.discord.token` | string | `""` | 是 |
| `channels.slack.bot_token` | string | `""` | 是 |
| `channels.slack.app_token` | string | `""` | 是 |
| `channels.qq.app_id` | string | `""` | 是 |
| `channels.qq.app_secret` | string | `""` | 是 |
| `channels.dingtalk.client_id` | string | `""` | 是 |
| `channels.dingtalk.client_secret` | string | `""` | 是 |
| `channels.maixcam.host` | string | `"0.0.0.0"` | 是 |
| `channels.maixcam.port` | int | `9000` | `1..65535` |

说明：`allow_from` 里的值是“渠道原生发送者 ID”，例如 Telegram 用户数字 ID、Slack 用户 ID、Discord 作者 ID。

## 5.4 `providers.*`

每个 provider 都支持：

- `api_key`
- `secret_key`
- `base_url`

Provider 细节：

- OpenRouter 固定地址：`https://openrouter.ai/api/v1`
- Claude 固定地址：`https://api.anthropic.com/v1`
- OpenAI 仅在设置了 `providers.openai.base_url` 时覆盖默认地址
- DeepSeek 固定地址：`https://api.deepseek.com/v1`
- Gemini 若 base_url 为空，默认 `https://generativelanguage.googleapis.com/v1beta/openai`
- Qwen 若 base_url 为空，默认 `https://dashscope.aliyuncs.com/compatible-mode/v1`
- Ollama 若 base_url 为空，默认 `http://localhost:11434`，内部自动拼接 `/v1`
- Ark 与 Qianfan 必须提供 `base_url`

Provider 选择逻辑：

1. 若模型名有前缀（如 `openai/...`、`claude/...`），优先按前缀选 provider。
2. 否则按顺序兜底：`openrouter -> claude -> openai -> deepseek -> gemini -> ark -> qianfan -> qwen -> ollama`。
3. 非 ollama provider 可使用配置中的 `api_key`，也可使用 `~/.golem/auth.json` 中 token。

## 5.5 `tools.*`

| 键 | 类型 | 默认值 | 规则 |
| --- | --- | --- | --- |
| `tools.exec.timeout` | int | `60` | 秒 |
| `tools.exec.restrict_to_workspace` | bool | `true` | 限制 `working_dir` 在工作区内 |
| `tools.web.search.api_key` | string | `""` | Brave key，可选 |
| `tools.web.search.max_results` | int | `5` | 运行时上限 `20` |
| `tools.voice.enabled` | bool | `false` | 启用入站音频转写 |
| `tools.voice.provider` | string | `openai` | 启用时必须是 `openai` |
| `tools.voice.model` | string | `gpt-4o-mini-transcribe` | OpenAI 兼容转写模型 |
| `tools.voice.timeout_seconds` | int | `30` | 非负；`0` 会回填为 `30` |

## 5.6 `gateway`、`heartbeat`、`log`

| 键 | 类型 | 默认值 | 规则 |
| --- | --- | --- | --- |
| `gateway.host` | string | `0.0.0.0` | 监听地址 |
| `gateway.port` | int | `18790` | 必须 `1..65535` |
| `gateway.token` | string | `""` | 设置后 `/chat` 必须携带 Bearer Token |
| `heartbeat.enabled` | bool | `true` | 是否启用心跳服务 |
| `heartbeat.interval` | int | `30` | 分钟，正值且小于 `5` 时会被提升到 `5` |
| `heartbeat.max_idle_minutes` | int | `720` | 超过该空闲阈值视为目标过期 |
| `log.level` | string | `info` | `debug`/`info`/`warn`/`error` |
| `log.file` | string | `""` | 设置后日志会追加写入该文件 |

## 6. 环境变量覆盖

Golem 支持 `GOLEM_` 前缀环境变量，示例：

```bash
export GOLEM_AGENTS_DEFAULTS_MODEL="openai/gpt-4o-mini"
export GOLEM_PROVIDERS_OPENAI_APIKEY="sk-..."
export GOLEM_GATEWAY_PORT=18790
export GOLEM_TOOLS_VOICE_ENABLED=true
export GOLEM_LOG_LEVEL=debug
```

建议使用“全大写 + 下划线”格式；嵌套键可直接展开（`agents.defaults.model` -> `GOLEM_AGENTS_DEFAULTS_MODEL`）。

## 7. CLI 命令总览

## 7.1 全局

```bash
golem --help
golem <command> --help
golem --log-level debug <command>
```

顶层命令：

- `auth`
- `channels`
- `chat`
- `completion`
- `cron`
- `init`
- `run`
- `skills`
- `status`

## 7.2 `golem init`

初始化配置、工作区和内置技能。

```bash
golem init
```

## 7.3 `golem chat [message]`

- 不带 message：启动 TUI 聊天
- 带 message：单次请求

```bash
golem chat
golem chat "总结最近日志"
```

## 7.4 `golem run`

启动以下组件：

- Agent Loop
- 消息总线路由
- 已启用渠道
- Gateway API
- Cron 服务
- Heartbeat 服务

```bash
golem run
```

## 7.5 `golem status`

输出配置、工作区、provider、工具、渠道、gateway、cron、skills 状态摘要。

```bash
golem status
```

## 7.6 `golem auth`

```bash
golem auth login --provider <name> --token <token>
golem auth login --provider openai --device-code
golem auth login --provider openai --browser
golem auth status
golem auth logout --provider openai
golem auth logout
```

说明：

- provider 支持：`openai|claude|openrouter|deepseek|gemini|ark|qianfan|qwen`（`anthropic` 会映射为 `claude`）
- OAuth 登录目前仅支持 `openai`

## 7.7 `golem channels`

```bash
golem channels list
golem channels status
golem channels start telegram
golem channels stop telegram
```

可用渠道名：

- `telegram`
- `whatsapp`
- `feishu`
- `discord`
- `slack`
- `qq`
- `dingtalk`
- `maixcam`

## 7.8 `golem cron`

```bash
golem cron list
golem cron add -n "hourly" -m "status report" --every 3600
golem cron add -n "daily" -m "briefing" --cron "0 9 * * *"
golem cron add -n "once" -m "reminder" --at "2026-02-14T09:00:00Z"
golem cron run <job_id>
golem cron enable <job_id>
golem cron disable <job_id>
golem cron remove <job_id>
```

## 7.9 `golem skills`

```bash
golem skills list
golem skills install owner/repo
golem skills install owner/repo/path/to/SKILL.md
golem skills show weather
golem skills search
golem skills search weather
golem skills remove weather
```

## 8. 内置工具（Agent）

默认注册工具如下：

| 工具名 | 关键参数 | 说明 |
| --- | --- | --- |
| `read_file` | `path`, `offset`, `limit` | 读取文件，可按行偏移/限制 |
| `write_file` | `path`, `content` | 覆盖写入文件 |
| `edit_file` | `path`, `old_text`, `new_text` | 仅替换唯一匹配片段 |
| `append_file` | `path`, `content` | 追加文件内容 |
| `list_dir` | `path` | 列目录 |
| `read_memory` | 无 | 读取 `memory/MEMORY.md` |
| `write_memory` | `content` | 写入长期记忆 |
| `append_diary` | `entry` | 追加每日日记 |
| `exec` | `command`, `working_dir` | 执行 shell，带超时和安全规则 |
| `web_search` | `query`, `max_results` | 有 Brave key 用 Brave，否则 DuckDuckGo 兜底 |
| `web_fetch` | `url`, `max_bytes` | 抓取网页并抽取文本，最大 1MB |
| `manage_cron` | `action` + 调度参数 | 管理 cron 任务 |
| `message` | `content`, `channel`, `chat_id` | 直接向渠道发送消息 |
| `spawn` | `task`, `label`, route 参数 | 异步子 Agent |
| `subagent` | `task`, `label`, route 参数 | 同步子 Agent |

安全边界：

- 文件工具和 `exec` 支持工作区路径边界检查。
- `exec` 会拦截高风险命令模式（如 `rm -rf /`、`mkfs`、fork bomb 等）。
- `edit_file` 要求 `old_text` 只能匹配一次；零匹配或多匹配都会拒绝。

## 9. 渠道与语音转写

## 9.1 渠道就绪条件

`golem run` 只会注册“已启用且凭据齐全”的渠道：

- Telegram：`token`
- WhatsApp：`bridge_url`
- Feishu：`app_id` + `app_secret`
- Discord：`token`
- Slack：`bot_token` + `app_token`
- QQ：`app_id` + `app_secret`
- DingTalk：`client_id` + `client_secret`
- MaixCam：`host` + `port`

## 9.2 语音转写规则

- 支持渠道：Telegram、Discord、Slack
- 必要条件：
  - `tools.voice.enabled=true`
  - `tools.voice.provider=openai`
  - OpenAI key（配置或 `golem auth login`）
- 转写失败时：
  - 不中断主流程
  - 自动回退占位文本（`[voice]` 或 `[audio: ...]`）

## 10. Gateway API

仅在 `golem run` 下可用：

- `GET /health`
- `GET /version`
- `POST /chat`

## 10.1 `POST /chat` 请求体

```json
{
  "message": "总结最新日志",
  "session_id": "ops-room",
  "sender_id": "api-client"
}
```

规则：

- `message` 必填
- `session_id` 默认 `default`
- `sender_id` 默认 `api`
- 若配置了 `gateway.token`，请求头必须带：

```text
Authorization: Bearer <token>
```

示例：

```bash
curl -X POST "http://127.0.0.1:18790/chat" \
  -H "Content-Type: application/json" \
  -d '{"message":"ping","session_id":"s1","sender_id":"u1"}'
```

## 11. 认证体系（Auth）

认证文件：`~/.golem/auth.json`

每个 provider 存储字段：

- `access_token`
- `refresh_token`（可选）
- `provider`
- `auth_method`（`token` 或 `oauth`）
- `expires_at`（可选）

行为说明：

- 当配置中 `api_key` 为空时，provider 层会尝试使用 auth store token。
- OpenAI OAuth 凭据在临近过期时会尝试自动刷新。

## 12. 技能系统

技能加载优先级：

1. `<workspace>/skills`
2. `~/.golem/skills`
3. builtin skills 目录

相关环境变量：

- `GOLEM_BUILTIN_SKILLS_DIR`
- `GOLEM_SKILLS_INDEX_URL`
- `GOLEM_GITHUB_RAW_BASE_URL`

## 13. Cron 与 Heartbeat

## 13.1 Cron

- 存储文件：`<workspace>/cron/jobs.json`
- 调度类型：
  - `every`（秒级间隔）
  - `cron`（5 段表达式）
  - `at`（RFC3339 单次执行）

## 13.2 Heartbeat

- 定期向“最近活跃渠道/会话”发送探活消息。
- 消息格式：

```text
[heartbeat] status=<ok|degraded> summary=<text>
```

- 路由目标持久化文件：`<workspace>/state/heartbeat.json`
- 服务重启后自动恢复目标会话
- 若目标会话超过 `heartbeat.max_idle_minutes`，则跳过发送

## 14. 日常运维操作

## 14.1 日检命令

```bash
golem status
golem channels status
golem cron list
```

## 14.2 Smoke 测试

```bash
go test ./...
go run ./cmd/golem status
go run ./cmd/golem chat "ping"
```

## 14.3 常见问题速查

| 现象 | 常见原因 | 处理方式 |
| --- | --- | --- |
| `no provider configured` | 没有配置 key/token/base_url | 填 provider 配置或执行 `golem auth login` |
| 渠道显示 enabled 但 not ready | 渠道必要凭据缺失 | 补齐该渠道必填配置 |
| gateway `/chat` 返回 `401` | 缺失或错误 Bearer token | 修正 `Authorization` 请求头 |
| 无 heartbeat 消息 | 尚无活跃会话/心跳关闭/会话过期 | 先收一条普通消息并检查 heartbeat 配置 |
| 语音转写不生效 | 未启用 voice 或缺少 OpenAI 凭据 | 开启 `tools.voice` 并配置 OpenAI key/token |

## 15. 安全建议

- 保护好 `~/.golem/auth.json` 与 `~/.golem/config.json`。
- 对外暴露 Gateway 前务必配置 `gateway.token`。
- 在共享或高风险环境中保持 `tools.exec.restrict_to_workspace=true`。
- 为渠道配置 `allow_from`，避免未授权来源。

## 16. 相关文档

- 英文运维手册：`docs/operations/runbook.md`
- 中文运维手册：`docs/operations/runbook.zh-CN.md`
- 项目概览：`README.md`
