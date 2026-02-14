# Golem 运维 Runbook

## 适用范围

本手册覆盖以下场景的日常运维与一级故障响应：

- `golem run` 服务模式
- Gateway HTTP API（`/health`、`/version`、`/chat`）
- Heartbeat 服务（定时健康探测与回传）
- Telegram 渠道集成
- Provider/Model 连通性
- 工具执行与记忆子系统
- 容器化部署（`Dockerfile` / `docker-compose.yml`）

## 快速健康检查清单

1. 检查进程与配置状态：`golem status`
2. 检查 API 健康：`curl http://127.0.0.1:18790/health`
3. 检查版本接口：`curl http://127.0.0.1:18790/version`
4. 检查日志（若已配置 `log.file`），重点关注：
   - `request_id`
   - 工具执行耗时
   - 外发消息失败记录
   - heartbeat 派发状态

## 常见故障

### 1. 服务启动失败

症状：

- `golem run` 启动后立刻退出
- 报错中包含 config/gateway/provider 相关信息

处理步骤：

1. 校验配置文件 JSON 语法。
2. 确认 `gateway.port` 在 `1` 到 `65535` 之间。
3. 确认当前 workspace 模式对应路径有效。
4. 执行：
   - `go test ./...`
   - `go vet ./...`

### 2. `/chat` 返回 `401 unauthorized`

症状：

- Gateway 的 `/chat` 请求被拒绝

处理步骤：

1. 检查配置中是否设置了 `gateway.token`。
2. 若已设置，请在请求头带上：`Authorization: Bearer <token>`
3. 使用相同请求重试，并附带 `X-Request-ID` 便于追踪。

### 3. `/chat` 返回 `500 internal_error`

症状：

- Gateway 接收请求成功，但响应失败

处理步骤：

1. 从响应中获取 `request_id`。
2. 用 `request_id` 检索日志。
3. 检查 provider 凭据和目标端点连通性。
4. 核对配置中的模型名与 provider 映射关系。

### 4. 渠道消息无响应

症状：

- 渠道已连接，但不回复消息

处理步骤：

1. 确认目标渠道已启用（例如 `channels.telegram.enabled=true`）。
2. 校验渠道凭据和 `allow_from` 发送者白名单。
3. 检查日志中的以下错误：
   - 渠道初始化失败
   - 外发消息失败
4. 从白名单账号在该渠道发送测试消息。

### 5. 工具执行报错

症状：

- Agent 回复中出现工具执行错误

处理步骤：

1. 定位 `request_id` 并检查对应工具执行日志。
2. 确认工作区边界配置（`restrict_to_workspace`）。
3. 若是 web_search，检查 `tools.web.search.api_key`。
4. 若是 memory 工具，确认 `memory/MEMORY.md` 存在且可写。

### 6. Heartbeat 未送达

症状：

- 没有向最近活跃渠道/会话回传周期性心跳

处理步骤：

1. 确认 `~/.golem/config.json` 中 `heartbeat.enabled=true`。
2. 确认 `heartbeat.interval` 合理（默认 `30`，最小有效值 `5`，单位：分钟）。
3. 确认该渠道/会话最近有入站对话（heartbeat 只发往最近活跃会话）。
4. 在日志中检索：
   - `heartbeat service started`
   - `heartbeat dispatched`
   - `heartbeat run failed`
5. 视场景调整 `heartbeat.max_idle_minutes`。
   降低可收紧“最近活跃”判定，提高可覆盖长时间空闲场景。

## 日志排查建议

- `log.level` 建议：
  - 故障定位期使用 `debug`
  - 常规生产使用 `info`
  - 需要降噪时使用 `warn/error`
- 生产环境建议配置 `log.file`。
- 故障记录中始终保留 `request_id`。
- 工具相关故障建议关联 `request_id`、`channel`、`tool_duration`，快速定位慢路径或失败路径。

## 恢复流程

### 平滑重启

1. 发送停止信号（Ctrl+C / SIGTERM）。
2. 等待优雅退出完成。
3. 重新启动：`golem run`

### 回滚

1. 切换到最近稳定标签：`git checkout <tag>`
2. 构建并部署该版本。
3. 验证 `/health` 与 `/version`。

### 容器重启（docker-compose）

1. 拉取最新代码并重建：`docker compose build --no-cache`
2. 重启服务：`docker compose up -d`
3. 验证：
   - `curl http://127.0.0.1:18790/health`
   - `docker compose logs --tail=200 golem`

## 发布安全门禁

发布流程会在以下检查全部通过后才允许构建/发布：

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`

生产发布流程不要绕过这些检查。

## 容量建议

- Gateway 与渠道负载应与部署资源匹配，建议起步配置：
  - `2 vCPU`
  - `2-4 GB RAM`
  - 用于 `~/.golem` 状态目录的持久化磁盘
- 控制渠道外发压力：
  - `channel.Manager` 默认已启用有界并发发送
  - 持续监控各渠道发送失败峰值
- heartbeat 间隔建议 `>=5` 分钟，避免噪声回调和不必要的模型/工具压力。
- 若 cron 任务较多，检查 `enabled_jobs` 与 `next_run` 分布，避免同步触发尖峰。
