# Golem 运维 Runbook

## 适用范围

本手册覆盖以下场景的日常运维和一线故障处理：

- `golem run` 服务模式
- Gateway HTTP API（`/health`、`/version`、`/chat`）
- Heartbeat 服务（定时探测与回传）
- Telegram/Discord/Slack 等通道联动
- Provider/Model 连通性
- 工具执行与记忆子系统
- 容器化部署（`Dockerfile` / `docker-compose.yml`）

## 快速健康检查

1. 检查进程状态：`golem status`
2. 检查 API 健康：`curl http://127.0.0.1:18790/health`
3. 检查版本接口：`curl http://127.0.0.1:18790/version`
4. 检查日志重点：
   - `request_id`
   - tool 执行耗时
   - tool 错误/超时比例和 p95 代理延迟
   - outbound 发送失败
   - heartbeat 派发状态
5. 检查运行时指标快照文件：`<workspace>/state/runtime_metrics.json`

## 常见故障

### 1. 服务启动失败

症状：

- `golem run` 立即退出
- 报错涉及 config/gateway/provider

处理步骤：

1. 校验配置文件 JSON 语法。
2. 确认 `gateway.port` 位于 `1..65535`。
3. 确认 workspace 路径与当前模式匹配。
4. 运行：
   - `go test ./...`
   - `go vet ./...`

### 2. `/chat` 返回 `401 unauthorized`

处理步骤：

1. 检查是否设置了 `gateway.token`。
2. 若设置，携带请求头：`Authorization: Bearer <token>`。
3. 重试时附带 `X-Request-ID` 便于追踪。

### 3. `/chat` 返回 `500 internal_error`

处理步骤：

1. 从响应中提取 `request_id`。
2. 用 `request_id` 检索日志。
3. 检查 provider 凭据与网络连通性。
4. 核对模型名与 provider 映射。

### 4. 通道消息无响应

处理步骤：

1. 确认目标通道已启用（例如 `channels.telegram.enabled=true`）。
2. 校验通道凭据和 `allow_from`。
3. 检查日志中的初始化失败和 outbound 发送失败。
4. 用白名单账号发送测试消息。

### 5. 工具执行报错

处理步骤：

1. 按 `request_id` 定位具体调用。
2. 确认 `restrict_to_workspace` 等边界配置。
3. web 搜索异常时检查 `tools.web.search.api_key`。
4. memory 异常时检查 `memory/MEMORY.md` 可读写。

### 6. 启动出现高风险策略告警

处理步骤：

1. 检查 `policy.mode`、`policy.off_ttl`、`policy.allow_persistent_off`。
2. 生产建议使用：
   - `policy.mode=strict`（推荐），或
   - `policy.mode=off` + 有限 `policy.off_ttl`（仅维护窗口）。
3. 确认 `<workspace>/state/audit.jsonl` 中策略审计事件完整。

### 7. MCP 调用不稳定

处理步骤：

1. 检查降级原因和重连日志。
2. 验证远端 MCP 服务健康与延迟。
3. `http_sse` 场景确认代理未破坏 SSE。
4. `stdio` 场景检查 stderr 上下文。
5. 必要时先临时禁用故障 MCP，保留健康 MCP 服务。

### 8. Heartbeat 未送达

处理步骤：

1. 确认 `heartbeat.enabled=true`。
2. 确认 `heartbeat.interval` 合理（默认 `30`，有效最小值 `5` 分钟）。
3. 确认目标会话近期有入站活动。
4. 检索日志：
   - `heartbeat service started`
   - `heartbeat dispatched`
   - `heartbeat run failed`

## 日志与观测建议

- `log.level`：故障期用 `debug`，平时用 `info`，降噪可用 `warn/error`。
- 生产建议配置 `log.file`。
- 故障记录始终包含 `request_id`。
- 关联 `request_id + channel + tool_duration` 快速定位慢路径和失败路径。

## 恢复流程

### 平滑重启

1. 发送终止信号（Ctrl+C / SIGTERM）。
2. 等待优雅退出。
3. 重启：`golem run`

### 回滚

1. 使用标准回滚脚本：`bash scripts/ops/rollback.sh <tag>`
2. 部署脚本构建的回滚版本二进制。
3. 验证：
   - `curl -fsS http://127.0.0.1:18790/health`
   - `curl -fsS http://127.0.0.1:18790/version`
4. 脚本不可用时手动执行：
   - `git checkout <tag>`
   - `go build -o golem ./cmd/golem`

### 容器重启（docker-compose）

1. `docker compose build --no-cache`
2. `docker compose up -d`
3. 验证：
   - `curl http://127.0.0.1:18790/health`
   - `docker compose logs --tail=200 golem`

## 发布安全门禁

发布流程必须通过：

- `bash scripts/release/preflight.sh <tag>`
  - 覆盖 `go test ./... -count=1`
  - 覆盖 `go test -race ./... -count=1`
  - 覆盖 `go vet ./...`
  - 校验语义化 tag 与 notes 模板存在
- `bash scripts/release/generate_notes.sh <tag> golem golem.exe release_notes.md`
  - 基于模板自动生成 changelog 与 checksums

## 容量建议

- 建议起步资源：
  - `2 vCPU`
  - `2-4 GB RAM`
  - 用于 `~/.golem` 的持久化磁盘
- 通道外发建议控制在 `channels.outbound` 策略内：
  - `max_concurrent_sends`
  - `retry_max_attempts`
  - `rate_limit_per_second`
  - `dedup_window_seconds`
- heartbeat 间隔建议 `>=5` 分钟。
- cron 任务多时注意 `enabled_jobs` 和 `next_run` 的分布，避免同一时刻突发。
