# Golem vs PicoClaw 缺陷对比清单（已按范围收敛）

## 本次范围约束
- 不纳入优化项：`Claude CLI/Codex` 专用 provider 路径。
- 不纳入优化项：`migrate`（旧生态迁移命令）。
- 文档以下内容已按你的要求删除这两项。

## 当前缺陷清单（保留项）

| 优先级 | 缺陷项 | 影响 |
|---|---|---|
| P0 | 缺少认证体系（OAuth/Token 登录、刷新、状态管理） | 无法覆盖需要账号态或短期令牌刷新的真实场景 |
| P0 | 运行状态不持久化（Heartbeat 目标会话仅内存保存） | 重启后心跳消息投递上下文丢失 |
| P1 | 工具链编辑能力弱（无 `edit_file` / `append_file`） | 代码修改原子性与可控性弱于 PicoClaw |
| P1 | 多模态消息输入能力弱（语音/音频转写缺失） | Telegram/Discord/Slack 场景可用性下降 |
| P2 | 工程交付配套不足（无 `Makefile`、无 `config.example.json`） | 本地开发与部署上手成本更高 |

---

## 证据索引（关键代码）

### 1) 认证体系缺口（P0）
- PicoClaw 具备 auth 命令和存储：
  - `picoclaw/cmd/picoclaw/main.go:131`
  - `picoclaw/pkg/auth/oauth.go:45`
  - `picoclaw/pkg/auth/store.go:10`
- Golem 根命令当前仅：
  - `cmd/golem/commands/root.go:30`
  - `cmd/golem/commands/root.go:37`

### 2) Heartbeat 目标会话未持久化（P0）
- Golem 仅内存态：
  - `internal/heartbeat/service.go:32`
  - `internal/heartbeat/service.go:70`
- PicoClaw 使用持久化 state：
  - `picoclaw/pkg/state/state.go:67`
  - `picoclaw/pkg/heartbeat/service.go:61`

### 3) 文件编辑工具缺口（P1）
- Golem 默认工具不含 `edit_file`/`append_file`：
  - `internal/agent/loop.go:64`
  - `internal/agent/loop.go:81`
- PicoClaw 已注册两项能力：
  - `picoclaw/pkg/agent/loop.go:67`
  - `picoclaw/pkg/agent/loop.go:68`

### 4) 多模态输入缺口（P1）
- Golem Telegram 仅 text/caption：
  - `internal/channel/telegram/telegram.go:86`
  - `internal/channel/telegram/telegram.go:88`
- PicoClaw 已有音频/语音和转写：
  - `picoclaw/pkg/channels/telegram.go:237`
  - `picoclaw/pkg/channels/telegram.go:271`
  - `picoclaw/pkg/voice/transcriber.go:20`

### 5) 交付配套缺口（P2）
- Golem 缺失：`Makefile`、`config/config.example.json`
- PicoClaw 具备：
  - `picoclaw/Makefile`
  - `picoclaw/config/config.example.json`

---

## 渐进式可执行优化计划（完整）

## 阶段 0：基线与验收框架（0.5 天）

### 目标
建立统一验收口径，避免“做了但不可验证”。

### 执行项
1. 固化当前回归命令到文档：`go test ./...`、`golem status`、`golem chat "ping"`（本地模型条件允许时）。
2. 新增计划文档：`docs/plans/2026-02-14-golem-optimization-roadmap.md`（记录阶段目标与完成标准）。
3. 约定每阶段完成后都跑同一组 smoke tests。

### 验收标准
- 团队成员可按文档独立执行回归流程。

### 执行状态（已启动并落地）
- 阶段 0 产物已创建：`docs/plans/2026-02-14-golem-optimization-roadmap.md`
- 已固化统一 smoke tests：
  - `go test ./...`
  - `go run ./cmd/golem status`
  - `go run ./cmd/golem chat "ping"`
- 已记录本机基线执行结果（含退出码与结果摘要）。

---

## 阶段 1：补齐认证体系（P0，1~2 天）

### 目标
让 Golem 支持“登录-保存-状态-退出”闭环。

### 建议改动
1. 新增命令：`golem auth login/logout/status`。
2. 新增认证存储模块（建议路径）：`internal/auth/`，文件建议：
   - `store.go`：凭据读写（`~/.golem/auth.json`）
   - `oauth.go`：OAuth/device-code 登录与刷新
3. 在 provider 选择处接入 auth token 读取（只做通用 token 注入，不做 Claude CLI/Codex 专用分支）。
4. 为 auth 增加单测：读写、过期、刷新分支。

### 影响文件（建议）
- `cmd/golem/commands/root.go`
- `cmd/golem/commands/auth.go`（新）
- `internal/auth/store.go`（新）
- `internal/auth/oauth.go`（新）
- `internal/provider/provider.go`

### 验收标准
- `golem auth status` 可显示认证状态。
- 配置中无明文 API key 时，仍可通过登录态调用模型。

---

## 阶段 2：Heartbeat 会话持久化（P0，0.5~1 天）

### 目标
解决重启后心跳目标丢失问题。

### 建议改动
1. 新增 `internal/state/manager.go` 持久化 `last_channel`、`last_chat_id`。
2. 在 `internal/agent/loop.go` 的消息处理路径写入最新活动会话。
3. `internal/heartbeat/service.go` 优先读取持久化状态；内存态作为运行时加速缓存。
4. 异常容错：状态文件损坏时回退默认空状态，不阻断服务。

### 影响文件（建议）
- `internal/state/manager.go`（新）
- `internal/agent/loop.go`
- `internal/heartbeat/service.go`
- `internal/state/manager_test.go`（新）

### 验收标准
- 服务重启后，心跳仍能投递到最近活跃会话。

---

## 阶段 3：补齐编辑型工具（P1，1 天）

### 目标
增加最小侵入修改能力，降低全量重写风险。

### 建议改动
1. 新增工具：
   - `edit_file`（按 `old_text -> new_text` 精确替换）
   - `append_file`（安全追加）
2. 默认注册到 `RegisterDefaultTools`。
3. 保留 workspace 限制、路径校验、空内容防御。

### 影响文件（建议）
- `internal/tools/edit.go`（新）
- `internal/tools/edit_test.go`（新）
- `internal/agent/loop.go`

### 验收标准
- Agent 能在不覆盖全文件的情况下完成局部修改。
- 对超出 workspace 的路径请求返回明确错误。

---

## 阶段 4：多模态输入（先 Telegram，再 Discord/Slack）（P1，2~3 天）

### 目标
优先打通语音/音频输入链路，提升真实聊天场景可用性。

### 建议改动
1. 抽象转写接口：`internal/voice/transcriber.go`（接口 + 默认实现）。
2. Telegram 先落地：下载语音/音频文件并转写，注入到 inbound content。
3. 第二步复用到 Discord/Slack 附件音频。
4. 配置新增 `tools.voice` 或 `providers.<x>.transcription`，允许开关和超时控制。

### 影响文件（建议）
- `internal/voice/transcriber.go`（新）
- `internal/channel/telegram/telegram.go`
- `internal/channel/discord/discord.go`
- `internal/channel/slack/slack.go`
- `internal/config/config.go`

### 验收标准
- Telegram 语音消息可转写为文本并进入 Agent 流程。
- 转写失败不影响普通文本消息处理。

---

## 阶段 5：工程交付配套（P2，0.5 天）

### 目标
降低新成员与 CI 接入成本。

### 建议改动
1. 新增 `Makefile`：`build/test/lint/run`。
2. 新增 `config/config.example.json`（最小可运行模板）。
3. README 增加“2 分钟启动”与 `make` 命令示例。

### 影响文件（建议）
- `Makefile`（新）
- `config/config.example.json`（新）
- `README.md`

### 验收标准
- 全新环境按 README + example config 可在 2~5 分钟内跑通。

---

## 执行节奏建议（推荐）
1. 第 1 周：阶段 0 + 阶段 1 + 阶段 2（先把 P0 清零）。
2. 第 2 周：阶段 3 + 阶段 4（能力增强）。
3. 第 3 周：阶段 5 + 文档收口 + 回归加固。

## 每阶段统一“完成定义”
- 代码：功能 + 单测齐全。
- 回归：`go test ./...` 通过。
- 文档：README/配置示例同步更新。
- 风险：已记录回滚方案（开关/配置降级/功能禁用）。

## 暂不做项（明确冻结）
- `Claude CLI/Codex` 专用 provider 路径。
- `golem migrate` 迁移命令。
