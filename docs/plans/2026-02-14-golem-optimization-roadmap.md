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
