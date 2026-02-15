# Golem User Guide (English)

Chinese version: [docs/user-guide.zh-CN.md](user-guide.zh-CN.md)

This document is the complete operator and user guide for Golem. It covers all current configuration keys, features, and operations from initialization to production-style running.

## 1. What Golem Is

Golem is a terminal-first personal AI assistant built with Go and Eino. It supports:

- Interactive chat (`golem chat`)
- Long-running multi-channel service (`golem run`)
- Tool-using agent loop (file, shell, memory, web, cron, messaging, subagent)
- Policy guard modes (`strict`/`relaxed`/`off`) with optional off TTL rollback
- Approval workflow for high-risk tools (`golem approval list|approve|reject`)
- MCP dynamic tool registration (`mcp.<server>.<tool>`) with degraded-server isolation
- Audit trail for policy decisions and tool execution
- Skills system (workspace/global/builtin)
- Gateway HTTP API
- Auth store with token/OAuth login
- Voice transcription for Telegram/Discord/Slack audio input
- Heartbeat probe delivery to latest active session

## 2. Install and Upgrade

## 2.1 Install from source

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

## 2.2 Build locally

```bash
go build -o golem ./cmd/golem
```

## 2.3 Verify install

```bash
golem --help
golem status
```

## 3. First-Time Setup and File Layout

## 3.1 Initialize

```bash
golem init
```

This creates config/workspace basics and builtin skills.

## 3.2 Important paths

| Path | Purpose |
| --- | --- |
| `~/.golem/config.json` | Main config file |
| `~/.golem/auth.json` | Provider auth credentials store |
| `~/.golem/builtin-skills/` | Builtin skills written by `golem init` |
| `<workspace>/memory/MEMORY.md` | Long-term memory |
| `<workspace>/memory/YYYY-MM-DD.md` | Daily diary files |
| `<workspace>/skills/` | Workspace skills |
| `<workspace>/sessions/*.jsonl` | Session history persistence |
| `<workspace>/cron/jobs.json` | Cron job store |
| `<workspace>/state/heartbeat.json` | Persisted latest heartbeat target |
| `<workspace>/state/approvals.json` | Approval request store |
| `<workspace>/state/audit.jsonl` | Append-only audit trail |
| `<workspace>/state/runtime_metrics.json` | Runtime metrics snapshot (tool/channel ratios and latency summary) |

`<workspace>` is resolved by `agents.defaults.workspace_mode`:

- `default`: `~/.golem/workspace`
- `cwd`: current working directory
- `path`: `agents.defaults.workspace`

## 4. Quick Start (Minimal)

```bash
golem init
cp config/config.example.json ~/.golem/config.json
cp .env.example .env.local
# edit key(s), for example providers.openai.api_key
golem status
golem chat "ping"
```

At minimum, fill one provider key (or run `golem auth login`) and set `GOLEM_GATEWAY_TOKEN` for network-exposed staging/production deployments.

Server mode:

```bash
golem run
```

## 5. Full Configuration Reference

Main file: `~/.golem/config.json`

## 5.1 Complete config example

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

| Key | Type | Default | Rules |
| --- | --- | --- | --- |
| `workspace_mode` | string | `default` | `default`/`cwd`/`path` |
| `workspace` | string | `~/.golem/workspace` | required when mode=`path` |
| `model` | string | `anthropic/claude-sonnet-4-5` | provider prefix affects provider selection |
| `max_tokens` | int | `8192` | must be `> 0` |
| `temperature` | float | `0.7` | must be in `[0, 2.0]` |
| `max_tool_iterations` | int | `20` | non-negative; `0` resets to `20` |

## 5.3 `channels.*`

| Key | Type | Default | Required when enabled |
| --- | --- | --- | --- |
| `channels.telegram.enabled` | bool | `false` | - |
| `channels.telegram.token` | string | `""` | yes |
| `channels.telegram.allow_from` | array | `[]` | optional sender allowlist |
| `channels.whatsapp.bridge_url` | string | `""` | yes |
| `channels.feishu.app_id` | string | `""` | yes |
| `channels.feishu.app_secret` | string | `""` | yes |
| `channels.feishu.encrypt_key` | string | `""` | optional |
| `channels.feishu.verification_token` | string | `""` | optional |
| `channels.discord.token` | string | `""` | yes |
| `channels.slack.bot_token` | string | `""` | yes |
| `channels.slack.app_token` | string | `""` | yes |
| `channels.qq.app_id` | string | `""` | yes |
| `channels.qq.app_secret` | string | `""` | yes |
| `channels.dingtalk.client_id` | string | `""` | yes |
| `channels.dingtalk.client_secret` | string | `""` | yes |
| `channels.maixcam.host` | string | `"0.0.0.0"` | yes |
| `channels.maixcam.port` | int | `9000` | `1..65535` |

Note: `allow_from` value format is channel-specific sender ID (for example Telegram numeric user id, Slack user id, Discord author id).

## 5.4 `providers.*`

Each provider block has:

- `api_key`
- `secret_key`
- `base_url`

Provider behavior:

- OpenRouter base URL is fixed to `https://openrouter.ai/api/v1`.
- Claude base URL is fixed to `https://api.anthropic.com/v1`.
- OpenAI uses `providers.openai.base_url` only when non-empty.
- DeepSeek base URL is fixed to `https://api.deepseek.com/v1`.
- Gemini defaults to `https://generativelanguage.googleapis.com/v1beta/openai` when base URL is empty.
- Qwen defaults to `https://dashscope.aliyuncs.com/compatible-mode/v1` when base URL is empty.
- Ollama defaults to `http://localhost:11434` and Golem appends `/v1`.
- Ark and Qianfan require `base_url`.

Provider selection logic:

1. If model has prefix (for example `openai/...`, `claude/...`, `qwen/...`), Golem tries that provider first.
2. Otherwise it falls back in order: `openrouter -> claude -> openai -> deepseek -> gemini -> ark -> qianfan -> qwen -> ollama`.
3. Non-ollama provider can use either `api_key` from config or token from `~/.golem/auth.json`.

## 5.5 `tools.*`

| Key | Type | Default | Rules |
| --- | --- | --- | --- |
| `tools.exec.timeout` | int | `60` | seconds |
| `tools.exec.restrict_to_workspace` | bool | `true` | blocks out-of-workspace `working_dir` |
| `tools.web.search.api_key` | string | `""` | Brave key optional |
| `tools.web.search.max_results` | int | `5` | runtime capped at `20` |
| `tools.voice.enabled` | bool | `false` | enables inbound audio transcription |
| `tools.voice.provider` | string | `openai` | when enabled, must be `openai` |
| `tools.voice.model` | string | `gpt-4o-mini-transcribe` | OpenAI-compatible model |
| `tools.voice.timeout_seconds` | int | `30` | non-negative; `0` resets to `30` |

## 5.6 `policy`, `mcp`

| Key | Type | Default | Rules |
| --- | --- | --- | --- |
| `policy.mode` | string | `strict` | one of `strict`/`relaxed`/`off` |
| `policy.off_ttl` | string | `""` | duration (for example `30m`); when set with mode `off`, auto-reverts to strict after ttl |
| `policy.allow_persistent_off` | bool | `false` | must be `true` to allow `mode=off` without `off_ttl` |
| `policy.require_approval` | array | `[]` | tool names requiring approval in strict mode |
| `mcp.servers.<name>.transport` | string | - | `stdio` or `http_sse` |
| `mcp.servers.<name>.command` | string | - | required for `stdio` transport |
| `mcp.servers.<name>.args` | array | `[]` | optional command args for `stdio` |
| `mcp.servers.<name>.env` | object | `{}` | optional env map for `stdio` |
| `mcp.servers.<name>.url` | string | - | required for `http_sse` transport |
| `mcp.servers.<name>.headers` | object | `{}` | optional headers for `http_sse` |

Notes:

- In strict mode, a blocked call creates/uses approval requests and returns pending status until approved.
- `off_ttl` is recommended for temporary maintenance windows; when ttl expires, strict mode is restored automatically.
- Startup emits explicit policy audit events (`policy_startup`, `policy_startup_persistent_off`) to `<workspace>/state/audit.jsonl`.
- If `policy.mode=off` without `off_ttl`, startup writes a high-risk warning in logs and audit trail.
- MCP server failures are isolated as degraded state; healthy servers still load.
- MCP call path has bounded retry/reconnect behavior for transient failures (HTTP/SSE retry, manager reconnect).

## 5.7 `gateway`, `heartbeat`, `log`

| Key | Type | Default | Rules |
| --- | --- | --- | --- |
| `gateway.host` | string | `0.0.0.0` | listen host |
| `gateway.port` | int | `18790` | `1..65535` |
| `gateway.token` | string | `""` | if set, `/chat` requires Bearer token |
| `heartbeat.enabled` | bool | `true` | toggles heartbeat service |
| `heartbeat.interval` | int | `30` | minutes, min clamp to `5` when positive |
| `heartbeat.max_idle_minutes` | int | `720` | skip stale sessions after threshold |
| `log.level` | string | `info` | `debug`/`info`/`warn`/`error` |
| `log.file` | string | `""` | when set, logs are appended to this file |

## 6. Environment Variable Overrides

Golem uses `GOLEM_` prefixed env vars. Examples:

```bash
export GOLEM_AGENTS_DEFAULTS_MODEL="openai/gpt-4o-mini"
export GOLEM_PROVIDERS_OPENAI_APIKEY="sk-..."
export GOLEM_GATEWAY_PORT=18790
export GOLEM_TOOLS_VOICE_ENABLED=true
export GOLEM_LOG_LEVEL=debug
```

Use uppercase + underscores; nested keys map naturally (`agents.defaults.model` -> `GOLEM_AGENTS_DEFAULTS_MODEL`).

Recommended environment split:

- `.env.local`
- `.env.staging`
- `.env.production`

Start from `.env.example`, keep `policy.mode=strict` and `policy.allow_persistent_off=false` as defaults, and only loosen with explicit TTL windows.

## 7. CLI Command Reference

## 7.1 Global

```bash
golem --help
golem <command> --help
golem --log-level debug <command>
```

Top-level commands:

- `auth`
- `channels`
- `chat`
- `completion`
- `cron`
- `approval`
- `init`
- `run`
- `skills`
- `status`

## 7.2 `golem init`

Initialize config/workspace/builtin skills.

```bash
golem init
```

## 7.3 `golem chat [message]`

- No message: starts TUI chat.
- With message: one-shot chat request.

```bash
golem chat
golem chat "Summarize recent logs"
```

## 7.4 `golem run`

Starts:

- Agent loop
- Message bus routing
- Enabled channels
- Gateway API server
- Cron service
- Heartbeat service

```bash
golem run
```

## 7.5 `golem status`

Prints config/workspace/provider/tool/channel/gateway/cron/skills status and runtime metrics summary.

```bash
golem status
```

Runtime metrics section includes:

- `tool_total`
- `tool_error_ratio`
- `tool_timeout_ratio`
- `tool_p95_proxy_ms`
- `channel_send_failure_ratio`

## 7.6 `golem auth`

```bash
golem auth login --provider <name> --token <token>
golem auth login --provider openai --device-code
golem auth login --provider openai --browser
golem auth status
golem auth logout --provider openai
golem auth logout
```

Notes:

- Provider names: `openai|claude|openrouter|deepseek|gemini|ark|qianfan|qwen` (and `anthropic` alias for `claude`)
- OAuth flow currently supported for `openai` only

## 7.7 `golem channels`

```bash
golem channels list
golem channels status
golem channels start telegram
golem channels stop telegram
```

Supported names:

- `telegram`
- `whatsapp`
- `feishu`
- `discord`
- `slack`
- `qq`
- `dingtalk`
- `maixcam`

## 7.8 `golem approval`

```bash
golem approval list
golem approval approve <id> --by <name> [--note <text>]
golem approval reject <id> --by <name> [--note <text>]
```

Notes:

- `list` shows pending approval requests.
- `approve` and `reject` require `--by` for decision attribution.
- Approval records are stored in `<workspace>/state/approvals.json`.

## 7.9 `golem cron`

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

## 7.10 `golem skills`

```bash
golem skills list
golem skills install owner/repo
golem skills install owner/repo/path/to/SKILL.md
golem skills show weather
golem skills search
golem skills search weather
golem skills remove weather
```

## 8. Built-in Tools (Agent)

Registered by default:

| Tool | Core arguments | Behavior |
| --- | --- | --- |
| `read_file` | `path`, `offset`, `limit` | Reads file content with optional line slice |
| `write_file` | `path`, `content` | Overwrites file content |
| `edit_file` | `path`, `old_text`, `new_text` | Replaces exactly one unique match |
| `append_file` | `path`, `content` | Appends content to file |
| `list_dir` | `path` | Lists directory entries |
| `read_memory` | none | Reads `memory/MEMORY.md` |
| `write_memory` | `content` | Writes long-term memory |
| `append_diary` | `entry` | Appends dated diary line |
| `exec` | `command`, `working_dir` | Runs shell command with timeout and safety checks |
| `web_search` | `query`, `max_results` | Brave search if key exists, else DuckDuckGo fallback |
| `web_fetch` | `url`, `max_bytes` | Fetches URL, strips HTML text, 1MB max cap |
| `manage_cron` | `action`, schedule fields | Creates/lists/enables/disables/removes jobs |
| `message` | `content`, `channel`, `chat_id` | Sends direct outbound message |
| `spawn` | `task`, `label`, route fields | Async subagent task |
| `subagent` | `task`, `label`, route fields | Sync subagent task |
| `mcp.<server>.<tool>` | MCP tool-specific JSON args | Dynamically registered from healthy MCP servers |

Safety boundaries:

- File tools and `exec` can enforce workspace boundary.
- `exec` blocks known dangerous patterns (`rm -rf /`, `mkfs`, fork bomb style, etc).
- `edit_file` requires unique `old_text` match; refuses 0 or multi-match edits.
- Policy/approval guard runs before execution, including dynamically registered MCP tools.

## 9. Channels and Voice Transcription

## 9.1 Channel readiness checks

`golem run` only registers enabled channels that pass required credentials:

- Telegram: `token`
- WhatsApp: `bridge_url`
- Feishu: `app_id` + `app_secret`
- Discord: `token`
- Slack: `bot_token` + `app_token`
- QQ: `app_id` + `app_secret`
- DingTalk: `client_id` + `client_secret`
- MaixCam: `host` + `port`

## 9.2 Voice transcription

- Supported inbound channels: Telegram, Discord, Slack
- Requires:
  - `tools.voice.enabled=true`
  - `tools.voice.provider=openai`
  - OpenAI API key in config or auth store token (`golem auth login`)
- On transcription failure:
  - Message processing continues
  - Fallback placeholder is inserted (`[voice]` or `[audio: ...]`)

## 10. Gateway API

Available in server mode (`golem run`):

- `GET /health`
- `GET /version`
- `POST /chat`

## 10.1 `POST /chat` request

```json
{
  "message": "Summarize latest logs",
  "session_id": "ops-room",
  "sender_id": "api-client"
}
```

Rules:

- `message` is required.
- `session_id` defaults to `default`.
- `sender_id` defaults to `api`.
- If `gateway.token` is set, include:

```text
Authorization: Bearer <token>
```

Example:

```bash
curl -X POST "http://127.0.0.1:18790/chat" \
  -H "Content-Type: application/json" \
  -d '{"message":"ping","session_id":"s1","sender_id":"u1"}'
```

## 11. Auth System

Credential file: `~/.golem/auth.json`

Stored fields per provider:

- `access_token`
- `refresh_token` (optional)
- `provider`
- `auth_method` (`token` or `oauth`)
- `expires_at` (optional)

Behavior:

- Provider layer can inject auth-store token when config key is empty.
- OpenAI OAuth credentials may auto-refresh when near expiry.

## 12. Skills System

Skill load precedence:

1. `<workspace>/skills`
2. `~/.golem/skills`
3. builtin skills directory

Related env vars:

- `GOLEM_BUILTIN_SKILLS_DIR`
- `GOLEM_SKILLS_INDEX_URL`
- `GOLEM_GITHUB_RAW_BASE_URL`

## 13. Cron and Heartbeat Services

## 13.1 Cron

- Store: `<workspace>/cron/jobs.json`
- Schedule kinds:
  - `every` (seconds interval)
  - `cron` (5-field expression)
  - `at` (RFC3339 one-shot)

## 13.2 Heartbeat

- Sends periodic probe messages to latest active channel/chat.
- Message format:

```text
[heartbeat] status=<ok|degraded> summary=<text>
```

- Persists routing target to `<workspace>/state/heartbeat.json`.
- Resumes persisted target after restart.
- Skips stale targets when idle exceeds `heartbeat.max_idle_minutes`.

## 14. Day-2 Operations

## 14.1 Daily checks

```bash
golem status
golem channels status
golem cron list
```

## 14.2 Smoke test set

```bash
go test ./...
go run ./cmd/golem status
go run ./cmd/golem chat "ping"
```

## 14.3 Troubleshooting quick map

| Symptom | Typical cause | Action |
| --- | --- | --- |
| `no provider configured` | no key/token/base URL | set provider config or run `golem auth login` |
| channel shows enabled but not ready | missing required channel credentials | fill channel config fields |
| gateway `/chat` returns `401` | missing/invalid bearer token | set correct `Authorization` header |
| no heartbeat output | no active session yet or heartbeat disabled/stale | send one normal message first, verify heartbeat config |
| voice transcription not active | voice disabled or missing OpenAI credentials | enable `tools.voice` and provide OpenAI key/token |

## 15. Security Notes

- Keep `~/.golem/auth.json` and `~/.golem/config.json` private.
- Set `gateway.token` before exposing gateway outside localhost.
- Keep `tools.exec.restrict_to_workspace=true` in shared or risky environments.
- Review channel `allow_from` to avoid unauthorized senders.

## 16. Related Docs

- English runbook: `docs/operations/runbook.md`
- Chinese runbook: `docs/operations/runbook.zh-CN.md`
- Main overview: `README.md`
