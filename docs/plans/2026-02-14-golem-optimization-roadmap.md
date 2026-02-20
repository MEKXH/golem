# Golem Optimization Roadmap (Stage 0 Baseline)

## Goal
- Freeze a single verification baseline for all subsequent phases.
- Make phase acceptance reproducible with the same smoke test suite.

## Scope Freeze
- Excluded by decision:
  - `Claude CLI/Codex` provider-specific path
  - `golem migrate` command

## Unified Smoke Test Suite

Run these commands after every phase merge:

1. `go test ./...`
2. `go run ./cmd/golem status`
3. `go run ./cmd/golem chat "ping"`

### Pass Criteria
- Command 1: exit code `0`.
- Command 2: exit code `0` and status output rendered normally.
- Command 3:
  - If at least one provider is configured: exit code `0` and returns assistant output.
  - If no provider is configured: explicit provider-missing error is accepted as expected baseline behavior.

## Baseline Evidence (Executed)

Execution date: `2026-02-15` (local environment time)

| Command | Exit Code | Result Summary |
|---|---:|---|
| `go test ./...` | 0 | All packages passed; no failing tests |
| `go run ./cmd/golem status` | 0 | Config/workspace OK; channels disabled; tools ready |
| `go run ./cmd/golem chat "ping"` | 0 | Returned `Pong` response |

## Stage 0 Checklist
- [x] Baseline commands executed with captured results.
- [x] Unified smoke suite documented for all next phases.
- [x] Acceptance criteria defined (including no-provider fallback).

## Phase Gate (Apply to Stage 1+)

Before claiming a phase complete:

1. Run the unified smoke suite.
2. Record outputs in the evidence log section below.
3. Confirm no regression versus baseline.

## Evidence Log Template

Copy this block for each phase:

```md
### Phase X Verification - YYYY-MM-DD
- `go test ./...` -> <exit_code>, <summary>
- `go run ./cmd/golem status` -> <exit_code>, <summary>
- `go run ./cmd/golem chat "ping"` -> <exit_code>, <summary>
- Regression decision: PASS / FAIL
```

### Phase 1 Verification - 2026-02-15
- `go test ./...` -> 0, all packages passed
- `go run ./cmd/golem status` -> 0, status output rendered normally
- `go run ./cmd/golem chat "ping"` -> 0, returned `Pong` response
- Regression decision: PASS

### Phase 1 Batch Scope Delivered
- Added auth credential store: `internal/auth/store.go`
- Added OAuth/device-code login support: `internal/auth/oauth.go`, `internal/auth/pkce.go`
- Added auth CLI commands: `golem auth login/logout/status`
- Added provider fallback to stored auth token when config API key is empty
- Added OAuth refresh attempt on provider resolve for OAuth credentials nearing expiry (OpenAI path)

### Phase 2 Verification - 2026-02-15
- `go test ./...` -> 0, all packages passed
- `go run ./cmd/golem status` -> 0, status output rendered normally
- `go run ./cmd/golem chat "ping"` -> 0, returned `Pong` response
- Regression decision: PASS

### Phase 2 Batch Scope Delivered
- Added persisted heartbeat state manager: `internal/state/manager.go`
- Added state manager tests for round-trip, missing file, and corrupt file fallback
- Wired heartbeat service to hydrate last active session from disk on startup
- Wired heartbeat activity tracking to persist latest `channel/chat_id/seen_at` on each message
- Updated run command wiring to provide state manager to heartbeat service
- Added heartbeat tests for persisted-session restore and activity persistence behavior

### Phase 3 Verification - 2026-02-15
- `go test ./...` -> 0, all packages passed
- `go run ./cmd/golem status` -> 0, status output rendered normally
- `go run ./cmd/golem chat "ping"` -> 0, returned `Pong` response
- Regression decision: PASS

### Phase 3 Batch Scope Delivered
- Added `edit_file` and `append_file` tools: `internal/tools/edit.go`
- Added tool tests (replace, append, path boundary, empty-content defense): `internal/tools/edit_test.go`
- Registered new tools into default agent toolchain: `internal/agent/loop.go`
- Added registration coverage in agent tests and status visibility

### Phase 4 Verification - 2026-02-15
- `go test ./...` -> 0, all packages passed
- `go run ./cmd/golem status` -> 0, status output rendered normally
- `go run ./cmd/golem chat "ping"` -> 0, returned `Pong` response
- Regression decision: PASS

### Phase 4 Batch Scope Delivered
- Added voice transcription abstraction and OpenAI-compatible implementation: `internal/voice/transcriber.go`
- Added voice transcription tests (multipart request, success path, HTTP error path): `internal/voice/transcriber_test.go`
- Extended Telegram inbound handling for `voice` / `audio` messages with optional transcription: `internal/channel/telegram/telegram.go`
- Extended Discord inbound handling for audio attachments with optional transcription: `internal/channel/discord/discord.go`
- Extended Slack inbound handling for audio files with optional transcription: `internal/channel/slack/slack.go`
- Guaranteed transcription-failure fallback for normal text messages (no regression): `internal/channel/telegram/telegram_test.go`
- Added regression tests for Discord/Slack transcription fallback behavior
- Added configurable voice settings (`tools.voice.enabled/provider/model/timeout_seconds`): `internal/config/config.go`
- Wired transcriber construction in run path with OpenAI API key/auth-store fallback: `cmd/golem/commands/run.go`

### Phase 5 Verification - 2026-02-15
- `go test ./...` -> 0, all packages passed
- `go run ./cmd/golem status` -> 0, status output rendered normally
- `go run ./cmd/golem chat "ping"` -> 0, returned `Pong` response
- Regression decision: PASS

### Phase 5 Batch Scope Delivered
- Added engineering `Makefile` with standard targets: `build`, `test`, `test-race`, `lint`, `run`, `status`, `chat`, `smoke`
- Added runnable config template: `config/config.example.json`
- Updated `README.md` with 2-5 minute quick start, template-based config bootstrap, and `make`-based workflow
