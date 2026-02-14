# Golem Operations Runbook

## Scope

This runbook covers routine operations and first-response troubleshooting for:

- `golem run` server mode
- Gateway HTTP API (`/health`, `/version`, `/chat`)
- Heartbeat service (scheduled health probe and callback)
- Telegram channel integration
- Provider/model connectivity
- Tool execution and memory subsystem
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
   - outbound send failures
   - heartbeat dispatch status

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

### 6. Heartbeat Does Not Deliver Messages

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

1. Checkout last known good tag:
   `git checkout <tag>`
2. Build and deploy that version.
3. Verify `/health` and `/version`.

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

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`

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
