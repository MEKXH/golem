# Golem (גּוֹלֶם)

<div align="center">

<img src="docs/logo.png" width="180" />

[![Go Version](https://img.shields.io/github/go-mod/go-version/MEKXH/golem?style=flat-square&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/MEKXH/golem?style=flat-square&logo=github)](https://github.com/MEKXH/golem/releases/latest)
[![Listed on Shelldex](https://shelldex.com/badges/shelldex-badge.svg)](https://shelldex.com/projects/golem/)
[![License](https://img.shields.io/github/license/MEKXH/golem?style=flat-square)](LICENSE)

**The self-evolving GeoAI Agent — your private GIS analyst.**

</div>

Golem is a **vertical AI Agent for the geospatial industry**, built with [Go](https://go.dev/) and [Eino](https://github.com/cloudwego/eino). It bridges the gap between natural-language interaction and professional GIS workflows — turning complex GDAL/PostGIS operations into conversational requests accessible from a **WebUI**, **terminal TUI**, or **any IM channel**.

Unlike generic chatbot wrappers, Golem ships with a real agent loop (tool calling up to 20 iterations), workspace-local GDAL/PostGIS tooling, learned pipeline reuse, fabricated tool scaffolding, and skill telemetry loops — all governed by a built-in approval and audit framework.

> **Golem (גולם)**: In Jewish folklore, a golem is an animated being made from inanimate matter, created to serve.

## Documentation

- [README (简体中文)](README.zh-CN.md)
- [User Guide (English)](docs/user-guide.md)
- [使用手册（简体中文）](docs/user-guide.zh-CN.md)
- [Operations Runbook (English)](docs/operations/runbook.md)
- [运行手册（简体中文）](docs/operations/runbook.zh-CN.md)

## Why Golem

The geospatial industry suffers from **fragmented tooling, steep learning curves, and repetitive workflows**. Existing GeoAI solutions are either locked inside desktop GIS platforms (QGIS/ArcGIS plugins), limited to Jupyter notebooks, or lack autonomous execution capabilities.

Golem solves this with a unique combination:

| Pain Point                                | How Golem Addresses It                                                 |
| ----------------------------------------- | ---------------------------------------------------------------------- |
| GDAL commands are hard to memorize        | Natural-language → GDAL command orchestration                          |
| PostGIS spatial SQL is error-prone        | Verified Spatial SQL Codebook with pattern matching                    |
| CRS confusion is a daily nightmare        | Auto-detection, smart projection selection, common-mistake warnings    |
| Analysis workflows are repeated manually  | Learned pipeline reuse with parameter-aware replay                     |
| Missing tools require custom scripting    | Fabricated tool scaffolding — Agent generates new Geo tools at runtime |
| GIS tools are inaccessible to non-experts | Three access modes: WebUI / TUI / IM channels (Telegram, Slack, etc.)  |
| Tool execution risks on production data   | Built-in approval gate, policy enforcement, and audit trail            |

## Core Differentiators

```
┌─────────────────────────────────────────────────────────────────┐
│                     Golem Architecture                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Layer 4: Self-Evolution                                         │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐     │
│  │ Tool         │ │ Skill        │ │ Pipeline             │     │
│  │ Fabrication  │ │ Telemetry    │ │ Learning             │     │
│  │ (auto-gen    │ │ (track &     │ │ (replay successful   │     │
│  │  new tools)  │ │  improve)    │ │  Geo sequences)      │     │
│  └──────────────┘ └──────────────┘ └──────────────────────┘     │
│                                                                  │
│  Layer 3: Domain Knowledge                                       │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐     │
│  │ Spatial SQL  │ │ CRS          │ │ Data Catalog         │     │
│  │ Codebook     │ │ Intelligence │ │ Connector            │     │
│  └──────────────┘ └──────────────┘ └──────────────────────┘     │
│                                                                  │
│  Layer 2: GIS Tool Layer                                         │
│  ┌─────┐ ┌──────┐ ┌──────┐ ┌───────┐ ┌─────────────────┐      │
│  │GDAL │ │Post  │ │CRS   │ │Format │ │Data Catalog     │      │
│  │/OGR │ │GIS   │ │Detect│ │Convert│ │& SQL Codebook   │      │
│  └─────┘ └──────┘ └──────┘ └───────┘ └─────────────────┘      │
│                                                                  │
│  Layer 1: Agent Engine                                           │
│  ┌──────┐ ┌──────┐ ┌─────┐ ┌──────┐ ┌─────┐ ┌──────────┐      │
│  │Agent │ │Tools │ │Bus  │ │Cron  │ │Mem  │ │Approval  │      │
│  │ Loop │ │Reg   │ │     │ │      │ │     │ │& Audit   │      │
│  └──────┘ └──────┘ └─────┘ └──────┘ └─────┘ └──────────┘      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1. Geo-Native Toolset

When `tools.geo.enabled=true`, Golem registers a full workspace-local Geo execution surface:

| Tool                 | Description                                                           |
| -------------------- | --------------------------------------------------------------------- |
| `geo_info`           | Inspect spatial datasets — format, extent, layer count, feature count |
| `geo_process`        | GDAL/OGR processing — clip, merge, reproject, rasterize, etc.         |
| `geo_crs_detect`     | Auto-detect CRS from coordinate range, metadata, or EPSG inference    |
| `geo_format_convert` | Convert between Shapefile, GeoJSON, GeoPackage, GeoTIFF, etc.         |
| `geo_data_catalog`   | Discover datasets in workspace or remote catalogs                     |
| `geo_sql_codebook`   | Query verified spatial SQL patterns with parameter substitution       |
| `geo_spatial_query`  | Execute PostGIS spatial SQL (when `postgis_dsn` is configured)        |

Workspace conventions:

- `geo-codebook/` — reusable spatial SQL patterns
- `tools/geo/` — fabricated workspace Geo tools and dry-run scaffolds
- `pipelines/geo/` — learned Geo tool sequences
- GDAL is required for file-processing tools; PostGIS is optional.

### 2. Self-Evolution Capability

No existing GeoAI competitor offers all three of these capabilities simultaneously:

**Learned Pipeline Reuse** — When a Geo tool sequence succeeds, it is saved as a replay-ready pipeline under `pipelines/geo/`. On similar future requests, the Agent injects the learned sequence as a parameter-aware reuse candidate, reducing execution time and improving reliability.

**Fabricated Tool Scaffolding** — When the Agent encounters a spatial task with no matching tool, it generates a dry-run manifest/script bundle under `tools/geo/`. The scaffold passes validator checks and can be filled in by a human or by the Agent itself, then registered as a first-class Geo tool.

**Skill Telemetry** — Every Geo skill tracks `shown`, `selected`, `success`, and `failure` counters. A deterministic report view surfaces underperforming skills first, giving the Agent a local, explainable signal for continuous improvement.

### 3. Approval, Policy & Security Governance

Golem treats security as a first-class concern, not an afterthought:

| Capability                | Description                                                                                         |
| ------------------------- | --------------------------------------------------------------------------------------------------- |
| **Policy modes**          | `strict` / `relaxed` / `off` — control which tools require approval before execution                |
| **Approval gate**         | Sensitive tools (e.g. `exec`, `geo_spatial_query`) require explicit human approval in `strict` mode |
| **Temporary bypass**      | `off_ttl` allows time-limited policy relaxation that auto-reverts                                   |
| **Audit trail**           | All tool executions are logged to `state/audit.jsonl` with full request context                     |
| **Approval state**        | Pending/approved/rejected requests tracked in `state/approvals.json`                                |
| **CLI management**        | `golem approval list/approve/reject` for out-of-band approval workflows                             |
| **Workspace restriction** | Geo file operations and shell commands can be confined to workspace boundaries                      |
| **PostGIS readonly**      | Spatial queries default to read-only transactions when supported                                    |

This makes Golem suitable for production environments where spatial data involves privacy, commercial secrets, or regulatory requirements.

### 4. Three Access Modes — Lower the Barrier

Golem can be accessed through three modes, making professional GIS capabilities available to both technical and non-technical users:

```
┌──────────┐     ┌──────────┐     ┌──────────────────────────┐
│  WebUI   │     │ Terminal │     │   IM Channels            │
│  /       │     │ TUI      │     │  Telegram, Discord,      │
│  /console│     │ golem    │     │  Slack, Feishu, WhatsApp,│
│          │     │ chat     │     │  QQ, DingTalk, MaixCam   │
└────┬─────┘     └────┬─────┘     └──────────┬───────────────┘
     │                │                       │
     └────────────────┼───────────────────────┘
                      ▼
              ┌───────────────┐
              │  Golem Agent  │
              │  Engine       │
              └───────────────┘
```

- **WebUI** (`golem run`) — Landing page at `/`, chat console at `/console`. No install required for end users; share a URL and they can start spatial analysis immediately.
- **Terminal TUI** (`golem chat`) — Full-featured terminal interface for GIS professionals who prefer command-line workflows.
- **IM Channels** (`golem run`) — Connect Telegram, Discord, Slack, Feishu, WhatsApp, QQ, DingTalk, or MaixCam. Urban planners, field workers, and non-technical stakeholders can request spatial analysis from the apps they already use.

## Built-in Tools

| Tool                                                     | Description                                                                      |
| -------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `exec`                                                   | Run shell commands (workspace restriction supported)                             |
| `read_file` / `write_file` / `edit_file` / `append_file` | File read/write/edit/append in workspace                                         |
| `list_dir`                                               | List directory contents                                                          |
| `read_memory` / `write_memory`                           | Persistent memory access                                                         |
| `append_diary`                                           | Append daily notes                                                               |
| `web_search`                                             | Web search (Brave when API key exists; fallback available)                       |
| `web_fetch`                                              | Fetch and extract web page content                                               |
| `geo_*`                                                  | Geospatial toolset — GDAL/PostGIS workflows, CRS, format conversion, spatial SQL |
| `manage_cron`                                            | Manage scheduled jobs                                                            |
| `message`                                                | Send messages to channels                                                        |
| `spawn` / `subagent` / `workflow`                        | Delegate tasks to subagents and orchestrated workflows                           |

## Architecture Overview

### Core Components

| Component          | Path                  | Description                                                                      |
| ------------------ | --------------------- | -------------------------------------------------------------------------------- |
| **Agent Loop**     | `internal/agent/`     | Main processing loop with tool calling, max 20 iterations                        |
| **Message Bus**    | `internal/bus/`       | Event-driven message routing via Go channels                                     |
| **Channel System** | `internal/channel/`   | Multi-platform integrations (Telegram, Discord, Slack, etc.)                     |
| **Provider**       | `internal/provider/`  | Unified LLM interface via Eino's OpenAI wrapper                                  |
| **Session**        | `internal/session/`   | Persistent JSONL-based conversation history                                      |
| **Tools**          | `internal/tools/`     | Built-in tools: file, shell, memory, web, cron, geo, message, subagent, workflow |
| **Memory**         | `internal/memory/`    | Long-term memory and daily diary system                                          |
| **Skills**         | `internal/skills/`    | Extensible Markdown-based prompt packs                                           |
| **Cron**           | `internal/cron/`      | Scheduled job management                                                         |
| **Heartbeat**      | `internal/heartbeat/` | Periodic health probe and status reporting                                       |
| **Gateway**        | `internal/gateway/`   | HTTP API server plus embedded WebUI                                              |

### Data Flow

```
User Input (WebUI / TUI / Telegram / Discord / Slack...)
         │
         ▼
    Channel (receives & validates message)
         │
         ▼
    Bus.PublishInbound() ──▶ MessageBus.inbound
         │
         ▼
    Agent Loop (processes message)
         │
    ┌────┴────┐
    ▼         ▼         ▼
Session  Context  LLM Generate
(History) Builder  (with tools bound)
              │           │
              │           ▼
              │      Tools.Execute()
              │      (Geo tools, shell, file...)
              │           │
              └─────┬─────┘
                    ▼
         Bus.PublishOutbound()
                    │
                    ▼
         Channel Manager (routes)
                    │
                    ▼
         Channel.Send() ──▶ User
```

### Subagent System

Golem supports delegating tasks to subagents for parallel processing:

- **`spawn`**: Asynchronous subagent, returns task ID immediately, notifies via message bus
- **`subagent`**: Synchronous subagent, blocks until completion, returns result directly
- **`workflow`**: Built-in workflow orchestration (decompose task, run sequential/parallel subtasks, aggregate results)

### Memory System

Two-tier memory architecture:

1. **Long-term Memory**: Single `MEMORY.md` file for persistent knowledge
2. **Daily Diary**: `YYYY-MM-DD.md` files for timestamped journal entries

### LLM Providers

OpenRouter, Claude, OpenAI, DeepSeek, Gemini, Ark, Qianfan, Qwen, Ollama.

## Installation

### Option A: Download Binary

Download Windows/Linux binaries from [Releases](https://github.com/MEKXH/golem/releases).

### Option B: Install from Source

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

## Quick Start

### 1. Initialize config

```bash
golem init
```

This creates `~/.golem/config.json` and workspace directories.

### 2. Bootstrap with the example config

```bash
cp config/config.example.json ~/.golem/config.json
```

PowerShell:

```powershell
Copy-Item config/config.example.json "$HOME/.golem/config.json"
```

Then edit `~/.golem/config.json` and set at least one provider key (for example `providers.openai.api_key`).

To enable Geo tools, set `tools.geo.enabled` to `true` and optionally configure `gdal_bin_dir` and `postgis_dsn`.

Create an environment file from template:

```bash
cp .env.example .env.local
```

### 3. Start chatting

```bash
golem chat
```

One-shot spatial analysis:

```bash
golem chat "Inspect all shapefiles in this directory and report their CRS"
```

### 4. Start server mode (WebUI + IM channels)

```bash
golem run
```

Open:

- `http://127.0.0.1:18790/` for the landing page
- `http://127.0.0.1:18790/console` for the chat console

## CLI Commands

| Command                                         | Description                                  |
| ----------------------------------------------- | -------------------------------------------- |
| `golem init`                                    | Initialize config and workspace              |
| `golem chat [message]`                          | Start TUI chat or send one-shot message      |
| `golem run`                                     | Start server mode (WebUI + IM channels)      |
| `golem status [--json]`                         | Show system status summary                   |
| `golem auth login`                              | Save provider credentials via token or OAuth |
| `golem auth logout`                             | Remove provider credentials                  |
| `golem auth status`                             | Show current auth credential status          |
| `golem channels list/status/start/stop`         | Manage IM channels                           |
| `golem cron list/add/run/remove/enable/disable` | Manage scheduled jobs                        |
| `golem approval list/approve/reject`            | Manage tool execution approvals              |
| `golem skills list/install/remove/show/search`  | Manage skill packs                           |

## Configuration

Main file: `~/.golem/config.json`
Template: `config/config.example.json`

Key configuration sections:

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

`policy.mode` values:

- `strict`: enforce `require_approval` list before tool execution
- `relaxed`: allow execution without approval gate
- `off`: disable policy checks (use `off_ttl` for temporary bypass)

For the full configuration reference, see [User Guide](docs/user-guide.md).

### Environment Variables

All config keys support `GOLEM_` prefix:

```bash
export GOLEM_PROVIDERS_CLAUDE_APIKEY="your-key"
export GOLEM_TOOLS_GEO_ENABLED=true
export GOLEM_GATEWAY_TOKEN="your-token"
```

## Bootstrap Files

The agent's system prompt is built from these files (searched in workspace):

1. `IDENTITY.md` - Agent identity and persona
2. `SOUL.md` - Core beliefs and values
3. `USER.md` - User-specific context
4. `TOOLS.md` - Custom tool descriptions
5. `AGENTS.md` - Subagent definitions

## Operations

For incident handling, restart/rollback flow, and production guidance:

- [Operations Runbook (English)](docs/operations/runbook.md)
- [运行手册（简体中文）](docs/operations/runbook.zh-CN.md)

## Development

```bash
make build
make test
make lint
make smoke
```

Without `make`:

```bash
go test ./...
go test -race ./...
go vet ./...
go build -o golem ./cmd/golem
```

WebUI development:

```bash
npm --prefix web install
npm --prefix web run dev
npm --prefix web run build:gateway
```

## Roadmap

### Phase 1 — Geo Foundation (Completed)

- [x] Core Geo toolset: `geo_info`, `geo_process`, `geo_crs_detect`, `geo_format_convert`, `geo_data_catalog`, `geo_sql_codebook`
- [x] Optional PostGIS spatial query tool
- [x] Learned pipeline reuse with replay-ready hints
- [x] Dry-run fabricated tool scaffolding
- [x] Skill telemetry tracking and reporting
- [x] Approval gate and audit trail
- [x] Three access modes: WebUI, TUI, IM channels
- [x] Multi-provider LLM support (OpenRouter, Claude, OpenAI, DeepSeek, Gemini, etc.)

### Phase 2 — Domain Knowledge Deepening (In Progress)

- [ ] Expand Spatial SQL Codebook to 30+ verified PostGIS query patterns
- [ ] CRS intelligence: auto-projection selection for area/distance analysis, CGCS2000-vs-WGS84 warnings
- [ ] Data catalog connectors: OSM Overpass API, Sentinel STAC API, local filesystem scanning
- [ ] GIS-specific skill packs: spatial analysis, remote sensing, data ETL pipelines
- [ ] Geo tool fabrication v2: Agent-generated Python scripts auto-registered as workspace tools

### Phase 3 — Workflow Orchestration & Enterprise

- [ ] Pipeline orchestration: parameterized replay, conditional branching, failure recovery
- [ ] Spatial triggers for cron: "when imagery updates for this AOI, run change detection"
- [ ] Map/GeoJSON attachment delivery via IM channels
- [ ] Role-based spatial data access control
- [ ] Multi-agent tile-based parallel processing for large raster datasets
- [ ] Geo pipeline health dashboard in WebUI

### Phase 4 — Community & Ecosystem

- [ ] Community tool marketplace: share and install fabricated Geo tools via GitHub
- [ ] Multimodal spatial cognition: satellite imagery semantic parsing integrated into analysis workflows
- [ ] Industry vertical skill packs: urban planning, power grid inspection, transportation analysis
- [ ] Explainable spatial reasoning view: map animations and profile charts showing Agent's intermediate reasoning

## License

MIT
