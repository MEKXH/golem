# Golem (גּוֹלֶם)

<div align="center">

<img src="docs/logo.png" width="180" />

[![Go Version](https://img.shields.io/github/go-mod/go-version/MEKXH/golem?style=flat-square&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/MEKXH/golem?style=flat-square&logo=github)](https://github.com/MEKXH/golem/releases/latest)
[![CI Status](https://img.shields.io/github/actions/workflow/status/MEKXH/golem/ci.yml?style=flat-square&logo=github-actions)](https://github.com/MEKXH/golem/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/MEKXH/golem?style=flat-square)](LICENSE)

**会自我进化的时空数据分析 Agent —— 你的私人 GIS 分析师。**

</div>

Golem 是一个**面向地理信息行业的垂直 AI Agent**，基于 [Go](https://go.dev/) 和 [Eino](https://github.com/cloudwego/eino) 构建。它在自然语言交互与专业 GIS 工作流之间架起桥梁，将复杂的 GDAL/PostGIS 操作转化为对话式请求，通过 **WebUI**、**终端 TUI** 或**任意 IM 渠道**即可使用。

与通用聊天机器人问答助手不同，Golem 内置了真正的 Agent 循环、面向工作区的 GDAL/PostGIS 工具链、学习到的 pipeline 复用、fabricated tool 脚手架，以及 skill telemetry  —— 所有操作均受内置审批与审计框架治理。

> **Golem (גולם)**：在犹太传说中，Golem 是由无生命物质塑造并被赋予行动能力的"仆从"。

## 文档导航

- [README (English)](README.md)
- [使用手册（英文）](docs/user-guide.md)
- [使用手册（简体中文）](docs/user-guide.zh-CN.md)
- [运维手册（英文）](docs/operations/runbook.md)
- [运维手册（简体中文）](docs/operations/runbook.zh-CN.md)

## 为什么选择 Golem

地理信息行业长期受困于**工具碎片化、学习曲线陡峭、重复性工作流**。现有 GeoAI 方案要么锁死在桌面 GIS 平台内（QGIS/ArcGIS 插件），要么局限于 Jupyter Notebook，要么缺乏自主执行能力。

Golem 提供了独特的组合方案：

| 行业痛点 | Golem 的解决方式 |
|---|---|
| GDAL 命令难记难用 | 自然语言 → GDAL 命令编排 |
| PostGIS 空间 SQL 容易写错 | 经验证的空间 SQL Codebook，模式匹配优先 |
| 坐标系混乱是日常噩梦 | 自动检测 CRS、智能投影选择、常见错误预警 |
| 分析工作流反复手工执行 | learned pipeline 复用，参数化重放 |
| 缺失工具需要写脚本 | fabricated tool 脚手架 —— Agent 运行时自动生成新 Geo 工具 |
| GIS 工具对非专业人员门槛高 | 三种接入方式：WebUI / TUI / IM 渠道（Telegram、Slack 等） |
| 生产数据上的工具执行有风险 | 内置审批门控、策略执行和审计追踪 |

## 核心差异化能力

```
┌─────────────────────────────────────────────────────────────────┐
│                       Golem 架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  第四层：自主进化层                                                │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐     │
│  │ Tool         │ │ Skill        │ │ Pipeline             │     │
│  │ Fabrication  │ │ Telemetry    │ │ Learning             │     │
│  │ (自动生成    │ │ (跟踪并      │ │ (重放成功的          │     │
│  │  新工具)     │ │  持续改进)   │ │  Geo 序列)           │     │
│  └──────────────┘ └──────────────┘ └──────────────────────┘     │
│                                                                  │
│  第三层：领域知识层                                                │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐     │
│  │ Spatial SQL  │ │ CRS          │ │ Data Catalog         │     │
│  │ Codebook     │ │ Intelligence │ │ Connector            │     │
│  └──────────────┘ └──────────────┘ └──────────────────────┘     │
│                                                                  │
│  第二层：GIS 工具层                                               │
│  ┌─────┐ ┌──────┐ ┌──────┐ ┌───────┐ ┌─────────────────┐      │
│  │GDAL │ │Post  │ │CRS   │ │格式   │ │数据目录         │      │
│  │/OGR │ │GIS   │ │检测  │ │转换   │ │& SQL Codebook   │      │
│  └─────┘ └──────┘ └──────┘ └───────┘ └─────────────────┘      │
│                                                                  │
│  第一层：Agent 引擎                                               │
│  ┌──────┐ ┌──────┐ ┌─────┐ ┌──────┐ ┌─────┐ ┌──────────┐      │
│  │Agent │ │工具  │ │消息 │ │Cron  │ │记忆 │ │审批      │      │
│  │ 循环 │ │注册  │ │总线 │ │      │ │     │ │& 审计    │      │
│  └──────┘ └──────┘ └─────┘ └──────┘ └─────┘ └──────────┘      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1. Geo 原生工具集

当 `tools.geo.enabled=true` 时，Golem 注册一整套面向工作区的 Geo 执行面：

| 工具 | 说明 |
|---|---|
| `geo_info` | 检查空间数据集 —— 格式、范围、图层数、要素数 |
| `geo_process` | GDAL/OGR 处理 —— 裁剪、合并、重投影、栅格化等 |
| `geo_crs_detect` | 从坐标值范围、元数据或 EPSG 推断自动检测 CRS |
| `geo_format_convert` | Shapefile、GeoJSON、GeoPackage、GeoTIFF 等格式互转 |
| `geo_data_catalog` | 在工作区或远程目录中发现数据集 |
| `geo_sql_codebook` | 查询经验证的空间 SQL 模式，支持参数替换 |
| `geo_spatial_query` | 执行 PostGIS 空间 SQL（需配置 `postgis_dsn`） |

工作区约定：
- `geo-codebook/` —— 可复用空间 SQL 模式
- `tools/geo/` —— fabricated Geo 工具及 dry-run 脚手架
- `pipelines/geo/` —— 学习到的 Geo 工具序列
- 文件处理类 Geo 工具依赖 GDAL；PostGIS 为可选能力

### 2. 自我进化能力

现有 GeoAI 竞品中，没有一个同时具备以下三项能力：

**Learned Pipeline 复用** —— 当 Geo 工具序列执行成功后，会作为 replay-ready pipeline 保存到 `pipelines/geo/`。在后续相似请求中，Agent 将学到的序列以 parameter-aware reuse candidate 的形式注入 prompt，缩短执行时间并提升可靠性。

**Fabricated Tool 脚手架** —— 当 Agent 遇到没有匹配工具的空间任务时，会在 `tools/geo/` 下生成 dry-run manifest/script bundle。脚手架通过 validator 检查后，可由人或 Agent 补充实现，然后注册为一等 Geo 工具。

**Skill Telemetry** —— 每个 Geo skill 跟踪 `shown`、`selected`、`success`、`failure` 计数器。确定性的 report 视图将低表现技能排在前面，给 Agent 一个本地、可解释的持续改进信号。

### 3. 审批、策略与安全治理

Golem 将安全作为一等能力，而非事后补丁：

| 能力 | 说明 |
|---|---|
| **策略模式** | `strict` / `relaxed` / `off` —— 控制哪些工具在执行前需要审批 |
| **审批门控** | 敏感工具（如 `exec`、`geo_spatial_query`）在 `strict` 模式下需要人工显式审批 |
| **限时放开** | `off_ttl` 允许限时放宽策略，到期自动回收 |
| **审计追踪** | 所有工具执行记录写入 `state/audit.jsonl`，附带完整请求上下文 |
| **审批状态** | 待审批/已通过/已驳回请求记录在 `state/approvals.json` |
| **CLI 管理** | `golem approval list/approve/reject` 支持带外审批流 |
| **工作区限制** | Geo 文件操作和 Shell 命令可限制在工作区边界内 |
| **PostGIS 只读** | 空间查询默认使用只读事务 |

这使得 Golem 适用于空间数据涉及隐私、商业秘密或合规要求的生产环境。

### 4. 三种接入方式 —— 降低使用门槛

Golem 支持三种接入方式，让专业 GIS 能力同时服务技术人员和业务用户：

```
┌──────────┐     ┌──────────┐     ┌──────────────────────────┐
│  WebUI   │     │ 终端     │     │   IM 渠道                │
│  /       │     │ TUI      │     │  Telegram, Discord,      │
│  /console│     │ golem    │     │  Slack, 飞书, WhatsApp,  │
│          │     │ chat     │     │  QQ, 钉钉, MaixCam      │
└────┬─────┘     └────┬─────┘     └──────────┬───────────────┘
     │                │                       │
     └────────────────┼───────────────────────┘
                      ▼
              ┌───────────────┐
              │  Golem Agent  │
              │  引擎         │
              └───────────────┘
```

- **WebUI**（`golem run`）—— 首页 `/` 展示产品介绍，`/console` 提供聊天控制台。终端用户无需安装任何软件，分享一个 URL 即可开始空间分析。
- **终端 TUI**（`golem chat`）—— 全功能终端界面，适合偏好命令行工作流的 GIS 专业人员。
- **IM 渠道**（`golem run`）—— 接入 Telegram、Discord、Slack、飞书、WhatsApp、QQ、钉钉或 MaixCam。城市规划师、外业人员和非技术干系人，可以在他们日常使用的 App 中直接发起空间分析请求。

## 内置工具

| 工具 | 说明 |
|---|---|
| `exec` | 执行 Shell 命令（支持限制在工作区内） |
| `read_file` / `write_file` / `edit_file` / `append_file` | 在工作区中读取/写入/编辑/追加文件 |
| `list_dir` | 列出目录内容 |
| `read_memory` / `write_memory` | 读写长期记忆 |
| `append_diary` | 追加每日日志 |
| `web_search` | 网页搜索（有 Brave Key 优先使用 Brave） |
| `web_fetch` | 抓取并提取网页内容 |
| `geo_*` | 地理空间工具集 —— GDAL/PostGIS 工作流、CRS、格式转换、空间 SQL |
| `manage_cron` | 管理定时任务 |
| `message` | 向渠道发送消息 |
| `spawn` / `subagent` / `workflow` | 委托任务给子 Agent 与编排工作流 |

## 架构概览

### 核心组件

| 组件 | 路径 | 描述 |
|------|------|------|
| **Agent 循环** | `internal/agent/` | 主处理循环，支持工具调用，默认最多 20 轮 |
| **消息总线** | `internal/bus/` | 基于 Go Channel 的事件驱动消息路由 |
| **渠道系统** | `internal/channel/` | 多平台集成（Telegram、Discord、Slack 等） |
| **提供商** | `internal/provider/` | 通过 Eino OpenAI 封装层统一 LLM 接口 |
| **会话** | `internal/session/` | 持久化 JSONL 格式对话历史 |
| **工具** | `internal/tools/` | 内置工具：文件、Shell、记忆、网页、Cron、Geo、消息、子 Agent、workflow |
| **记忆** | `internal/memory/` | 长期记忆与每日日记系统 |
| **技能** | `internal/skills/` | 可扩展的 Markdown 提示词包 |
| **Cron** | `internal/cron/` | 定时任务管理 |
| **心跳** | `internal/heartbeat/` | 定期健康探测与状态回传 |
| **网关** | `internal/gateway/` | HTTP API 服务器与内嵌 WebUI |

### 数据流

```
用户输入 (WebUI / TUI / Telegram / Discord / Slack...)
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
              │      (Geo 工具、Shell、文件...)
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

### 子 Agent 系统

Golem 支持将任务委托给子 Agent 并行处理：

- **`spawn`**：异步子 Agent，立即返回任务 ID，通过消息总线通知结果
- **`subagent`**：同步子 Agent，阻塞直到完成，直接返回结果
- **`workflow`**：内置工作流编排（拆解任务、串/并行执行子任务、汇总每步结果）

### 记忆系统

双层记忆架构：

1. **长期记忆**：单个 `MEMORY.md` 文件存储持久知识
2. **每日日记**：`YYYY-MM-DD.md` 文件存储带时间戳的日记条目

### 支持的 LLM 提供商

OpenRouter、Claude、OpenAI、DeepSeek、Gemini、Ark、Qianfan、Qwen、Ollama。

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

```bash
cp config/config.example.json ~/.golem/config.json
```

PowerShell:

```powershell
Copy-Item config/config.example.json "$HOME/.golem/config.json"
```

然后编辑 `~/.golem/config.json`，至少填入一个 provider 的 key（例如 `providers.openai.api_key`）。

启用 Geo 工具需将 `tools.geo.enabled` 设为 `true`，并可选配置 `gdal_bin_dir` 和 `postgis_dsn`。

建议基于模板创建环境变量文件：

```bash
cp .env.example .env.local
```

### 3. 开始对话

```bash
golem chat
```

单次空间分析：

```bash
golem chat "检查当前目录下所有 Shapefile 的坐标系"
```

### 4. 启动服务模式（WebUI + IM 渠道）

```bash
golem run
```

访问：
- `http://127.0.0.1:18790/`：产品首页
- `http://127.0.0.1:18790/console`：聊天控制台

## CLI 命令总览

| 命令 | 说明 |
|---|---|
| `golem init` | 初始化配置和工作区 |
| `golem chat [message]` | 启动 TUI 对话或单次发送消息 |
| `golem run` | 启动服务模式（WebUI + IM 渠道） |
| `golem status [--json]` | 查看系统状态摘要 |
| `golem auth login/logout/status` | 管理 Provider 认证凭据 |
| `golem channels list/status/start/stop` | 管理 IM 渠道 |
| `golem cron list/add/run/remove/enable/disable` | 管理定时任务 |
| `golem approval list/approve/reject` | 管理工具执行审批 |
| `golem skills list/install/remove/show/search` | 管理技能包 |

## 配置说明

主配置文件：`~/.golem/config.json`
仓库模板文件：`config/config.example.json`

关键配置段：

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-sonnet-4-5",
      "max_tool_iterations": 20
    }
  },
  "tools": {
    "geo": {
      "enabled": true,
      "gdal_bin_dir": "",
      "restrict_to_workspace": true,
      "timeout_seconds": 120,
      "postgis_dsn": "",
      "query_timeout_seconds": 30,
      "max_rows": 200,
      "readonly": true
    }
  },
  "policy": {
    "mode": "strict",
    "require_approval": ["exec"]
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790,
    "token": ""
  }
}
```

`policy.mode` 可选值：

- `strict`：对 `require_approval` 中的工具强制走审批
- `relaxed`：允许执行，不触发审批
- `off`：关闭策略检查（建议搭配 `off_ttl` 仅限时放开）

完整配置参考请查看[使用手册](docs/user-guide.zh-CN.md)。

### 环境变量

所有配置项支持 `GOLEM_` 前缀：

```bash
export GOLEM_PROVIDERS_CLAUDE_APIKEY="your-key"
export GOLEM_TOOLS_GEO_ENABLED=true
export GOLEM_GATEWAY_TOKEN="your-token"
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

```bash
make build
make test
make lint
make smoke
```

如果本机没有 `make`：

```bash
go test ./...
go test -race ./...
go vet ./...
go build -o golem ./cmd/golem
```

WebUI 开发：

```bash
npm --prefix web install
npm --prefix web run dev
npm --prefix web run build:gateway
```

## 项目路线图

### 第一阶段 —— Geo 基础能力（已完成）

- [x] 核心 Geo 工具集：`geo_info`、`geo_process`、`geo_crs_detect`、`geo_format_convert`、`geo_data_catalog`、`geo_sql_codebook`
- [x] 可选 PostGIS 空间查询工具
- [x] learned pipeline 复用（replay-ready 提示）
- [x] dry-run fabricated tool 脚手架
- [x] skill telemetry 跟踪与报告
- [x] 审批门控与审计追踪
- [x] 三种接入方式：WebUI、TUI、IM 渠道
- [x] 多 Provider LLM 支持（OpenRouter、Claude、OpenAI、DeepSeek、Gemini 等）

### 第二阶段 —— 领域知识深化（进行中）

- [ ] 空间 SQL Codebook 扩充至 30+ 经验证的 PostGIS 查询模式
- [ ] CRS 智能化：面积/距离分析时的自动投影选择、CGCS2000 与 WGS84 混淆预警
- [ ] 数据目录连接器：OSM Overpass API、Sentinel STAC API、本地文件系统扫描
- [ ] GIS 专用技能包：空间分析、遥感影像处理、数据 ETL 管线
- [ ] Geo tool fabrication v2：Agent 生成的 Python 脚本自动注册为工作区工具

### 第三阶段 —— 工作流编排与企业级能力

- [ ] Pipeline 编排：参数化重放、条件分支、失败恢复
- [ ] 空间触发器 Cron：「当此 AOI 影像更新时，运行变化检测」
- [ ] 通过 IM 渠道投递地图/GeoJSON 附件
- [ ] 基于角色的空间数据访问控制
- [ ] 多 Agent 分幅并行处理大规模栅格数据
- [ ] WebUI 中的 Geo pipeline 健康看板

### 第四阶段 —— 社区与生态

- [ ] 社区工具市场：通过 GitHub 分享和安装 fabricated Geo 工具
- [ ] 多模态空间认知：卫星影像语义解析集成到分析工作流
- [ ] 行业垂直技能包：城市规划、电网巡检、交通分析
- [ ] 可解释的空间推理视图：地图动画和剖面图展示 Agent 中间推理过程

## 许可证

MIT
