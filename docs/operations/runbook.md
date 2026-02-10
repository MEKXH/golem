# Golem Operations Runbook

## Scope

This runbook covers routine operations and first-response troubleshooting for:

- `golem run` server mode
- Gateway HTTP API (`/health`, `/version`, `/chat`)
- Telegram channel integration
- Provider/model connectivity
- Tool execution and memory subsystem

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

### 4. Telegram Messages Not Responding

Symptoms:
- Bot online but no replies

Actions:
1. Confirm `channels.telegram.enabled=true`.
2. Verify bot token and `allow_from` sender IDs.
3. Check logs for:
   - telegram init failure
   - outbound send failure
4. Send a test message from an allowed account.

### 5. Tool Execution Errors

Symptoms:
- agent response contains tool errors

Actions:
1. Locate `request_id` and inspect tool execution logs.
2. Confirm workspace boundaries (`restrict_to_workspace`).
3. For web search, verify `tools.web.search.api_key`.
4. For memory tools, verify `memory/MEMORY.md` exists and is writable.

## Logging Guidance

- Use `log.level`:
  - `debug` during active incident investigation
  - `info` for normal production
  - `warn/error` for noise reduction
- Prefer setting `log.file` in production.
- Always record `request_id` in incident notes.

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

## Release Safety

Release workflow blocks build/release unless verification passes:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`

Do not bypass these checks in production release flow.
