# Golem Operations Runbook

## Scope

This runbook covers routine operations and first-response troubleshooting for Golem — a vertical AI Agent for the geospatial industry:

- `golem run` server mode
- Gateway HTTP API (`/health`, `/version`, `/chat`) and embedded WebUI
- Geo tool execution (GDAL/PostGIS workflows)
- Heartbeat service (scheduled health probe and callback)
- IM channel integrations (Telegram, Discord, Slack, etc.)
- Provider/model connectivity
- Tool execution, approval/audit, and memory subsystem
- Containerized deployment (`Dockerfile` / `docker-compose.yml`)

## Quick Health Checklist

1. Check process health:
   `golem status`
2. Check API health:
   `curl http://127.0.0.1:18790/health`
3. Check version endpoint:
   `curl http://127.0.0.1:18790/version`
4. Check logs (if `log.file` configured):
   - request ids
   - tool execution duration
   - tool error/timeout ratios and p95 proxy latency
   - outbound send failures
   - heartbeat dispatch status
5. Check runtime metrics snapshot file:
   - `<workspace>/state/runtime_metrics.json`

## Common Incidents

### 1. Server Fails to Start

Symptoms:
- `golem run` exits immediately
- startup error mentions config/gateway/provider

Actions:
1. Validate config file JSON syntax.
2. Confirm `gateway.port` is between `1` and `65535`.
3. Confirm workspace path is valid for current mode.
4. Run:
   - `go test ./...`
   - `go vet ./...`

### 2. `/chat` Returns `401 unauthorized`

Symptoms:
- Gateway `/chat` rejects requests

Actions:
1. Check whether `gateway.token` is set in config.
2. If set, send header:
   `Authorization: Bearer <token>`
3. Retry with the same request and include `X-Request-ID` for traceability.

### 3. `/chat` Returns `500 internal_error`

Symptoms:
- Gateway chat request accepted but response fails

Actions:
1. Locate `request_id` in response.
2. Search logs by `request_id`.
3. Check provider credentials and endpoint reachability.
4. Verify model name/provider mapping in config.

### 4. Channel Messages Not Responding

Symptoms:
- Channel connected but no replies

Actions:
1. Confirm target channel is enabled (e.g. `channels.telegram.enabled=true`).
2. Verify channel credentials and `allow_from` sender IDs.
3. Check logs for:
   - channel init failure
   - outbound send failure
4. Send a test message from an allowed account on that channel.

### 5. Tool Execution Errors

Symptoms:
- agent response contains tool errors

Actions:
1. Locate `request_id` and inspect tool execution logs.
2. Confirm workspace boundaries (`restrict_to_workspace`).
3. For web search, verify `tools.web.search.api_key`.
4. For memory tools, verify `memory/MEMORY.md` exists and is writable.

### 6. Policy High-Risk Warning at Startup

Symptoms:
- Startup logs show high-risk policy warning
- Audit log contains `policy_startup_persistent_off`

Actions:
1. Check `policy.mode`, `policy.off_ttl`, `policy.allow_persistent_off` in config.
2. For normal operations, switch to:
   - `policy.mode=strict` (recommended), or
   - `policy.mode=off` + finite `policy.off_ttl` (temporary maintenance only).
3. Confirm audit trail in `<workspace>/state/audit.jsonl` includes expected startup policy events.

### 7. MCP Tool Calls Become Unstable

Symptoms:
- intermittent MCP tool call failures
- degraded/reconnect messages in logs

Actions:
1. Check logs for server-level reconnect attempts and final degraded reason.
2. Verify remote MCP server health and endpoint latency.
3. For `http_sse`, verify gateway/proxy does not strip SSE semantics.
4. For `stdio`, inspect wrapped stderr context in logs for process bootstrap/runtime failures.
5. If unstable persists, disable the failing MCP server block temporarily and keep healthy servers active.

### 8. Heartbeat Does Not Deliver Messages

Symptoms:
- No periodic heartbeat callback to previously active channel/chat

Actions:
1. Confirm `heartbeat.enabled=true` in `~/.golem/config.json`.
2. Confirm `heartbeat.interval` is reasonable (default `30`, minimum effective value `5`, unit: minutes).
3. Confirm there was a recent inbound conversation on that channel/chat (heartbeat targets the latest active session).
4. Search logs for:
   - `heartbeat service started`
   - `heartbeat dispatched`
   - `heartbeat run failed`
5. If needed, reduce `heartbeat.max_idle_minutes` for stricter recency control or increase it for long-idle scenarios.

### 9. Geo Tool Execution Failures

Symptoms:
- `geo_process`, `geo_info`, `geo_crs_detect`, or `geo_format_convert` returns errors
- Agent reports "GDAL not found" or command timeout

Actions:
1. Confirm `tools.geo.enabled=true` in config.
2. Verify GDAL is installed and accessible: run `gdalinfo --version` on the host.
3. If GDAL is not on `$PATH`, set `tools.geo.gdal_bin_dir` to the directory containing GDAL executables.
4. Check `tools.geo.timeout_seconds` — large raster operations may need a higher value.
5. Verify `tools.geo.restrict_to_workspace` — if `true`, all input/output paths must be inside the workspace.
6. Check logs for workspace boundary violations or command whitelist rejections.

### 10. PostGIS Spatial Query Issues

Symptoms:
- `geo_spatial_query` tool is not registered
- Spatial queries return connection errors or timeouts

Actions:
1. Confirm `tools.geo.postgis_dsn` is set to a valid PostgreSQL/PostGIS connection string.
2. Test connectivity from the Golem host: `psql "<dsn>" -c "SELECT PostGIS_Version();"`.
3. For timeout issues, increase `tools.geo.query_timeout_seconds`.
4. For row-limit truncation, increase `tools.geo.max_rows`.
5. Confirm `tools.geo.readonly=true` if write access is not intended.
6. If using `policy.mode=strict`, ensure `geo_spatial_query` approval requests are handled via `golem approval approve`.

### 11. Fabricated Geo Tool or Pipeline Issues

Symptoms:
- Fabricated tools under `tools/geo/` fail validation
- Learned pipelines under `pipelines/geo/` are not matched for reuse

Actions:
1. Check `tools/geo/` for malformed manifest or script files.
2. Verify skill telemetry in `state/skill_telemetry.json` is being updated.
3. For pipeline matching issues, inspect `pipelines/geo/*.json` and confirm parameter patterns align with current requests.
4. Review logs for fabrication scaffold generation or pipeline hint injection messages.

## Logging Guidance

- Use `log.level`:
  - `debug` during active incident investigation
  - `info` for normal production
  - `warn/error` for noise reduction
- Prefer setting `log.file` in production.
- Always record `request_id` in incident notes.
- For tool incidents, correlate `request_id`, `channel`, and `tool_duration` to identify slow/failing paths quickly.

## Recovery Procedures

### Graceful Restart

1. Stop with signal (Ctrl+C / SIGTERM).
2. Wait for shutdown completion.
3. Start again:
   `golem run`

### Rollback

1. Run standard rollback script:
   `bash scripts/ops/rollback.sh <tag>`
2. Deploy the rebuilt binary from that tag.
3. Verify:
   - `curl -fsS http://127.0.0.1:18790/health`
   - `curl -fsS http://127.0.0.1:18790/version`
4. Manual fallback if script is unavailable:
   - `git checkout <tag>`
   - `go build -o golem ./cmd/golem`

### Container Restart (docker-compose)

1. Pull latest code and rebuild:
   `docker compose build --no-cache`
2. Restart service:
   `docker compose up -d`
3. Verify:
   - `curl http://127.0.0.1:18790/health`
   - `docker compose logs --tail=200 golem`

## Release Safety

Release workflow blocks build/release unless verification passes:

- `bash scripts/release/preflight.sh <tag>` (includes `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, semver tag check, notes template check)
- `bash scripts/release/generate_notes.sh <tag> golem golem.exe release_notes.md` (template-based release notes with changelog + checksums)

Do not bypass these checks in production release flow.

## Capacity Guidance

- Gateway and channel workloads should be bounded by deployment resources; start with:
  - `2 vCPU`
  - `2-4 GB RAM`
  - persistent disk for `~/.golem` state
- Keep channel outbound pressure controlled:
  - default bounded concurrent sends are enabled in `channel.Manager`
  - monitor send failure spikes per channel
- Use heartbeat interval `>=5` minutes to avoid noisy callbacks and unnecessary model/tool pressure.
- For heavy cron usage, review `enabled_jobs` and `next_run` distribution to avoid synchronized bursts.
