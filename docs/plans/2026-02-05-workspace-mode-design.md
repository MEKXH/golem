# Workspace Mode Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan.

**Goal:** Add a configurable workspace mode that can use default, cwd, or explicit path for all workspace-dependent features (sessions, context, exec).

**Architecture:** Introduce `agents.defaults.workspace_mode` with values `default | cwd | path`. Centralize logic in `Config.WorkspacePath()` so all components use the same resolved path. Update status output and add tests.

**Tech Stack:** Go, Viper/mapstructure, Cobra

---

## Design Summary

### Config & Resolution
- New config key: `agents.defaults.workspace_mode`
- Values: `default | cwd | path`
- Semantics:
  - `default`: current behavior (if `workspace` empty => `~/.golem/workspace`, otherwise use provided `workspace`)
  - `cwd`: use `os.Getwd()` as workspace root
  - `path`: require `workspace` to be set; use it

### Components Using Workspace
All of these will continue to use `Config.WorkspacePath()`:
- `session.NewManager(workspacePath)`
- `NewContextBuilder(workspacePath)`
- `tools.NewExecTool(..., workspacePath)`

### Error Handling
- `workspace_mode=path` and empty `workspace` => return error
- `workspace_mode=cwd` and `os.Getwd()` fails => return error
- `workspace_mode=default` => never error (compat)

### Testing
- Unit tests for `WorkspacePath()` covering all modes
- Status command prints `workspace_mode`

---

### Task 1: Add config fields + default

**Files:**
- Modify: `internal/config/config.go`
- Modify: `cmd/golem/commands/status.go`

**Step 1: Write failing test**

Add tests to `internal/config/config_test.go`:
```go
func TestWorkspacePath_Default(t *testing.T) {
    cfg := DefaultConfig()
    cfg.Agents.Defaults.Workspace = ""
    cfg.Agents.Defaults.WorkspaceMode = "default"
    got := cfg.WorkspacePath()
    want := filepath.Join(ConfigDir(), "workspace")
    if got != want { t.Fatalf("got %s want %s", got, want) }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/...`
Expected: FAIL (unknown field or behavior)

**Step 3: Minimal implementation**

- Add `WorkspaceMode string \`mapstructure:"workspace_mode"\`` to `AgentDefaults`
- Set default to `"default"`
- Update `WorkspacePath()` logic for mode handling

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
 git commit -m "feat: add workspace mode" 
```

---

### Task 2: Status output includes workspace mode

**Files:**
- Modify: `cmd/golem/commands/status.go`

**Step 1: Write failing test**

Add/extend test in `cmd/golem/commands/status_test.go` (create if missing):
```go
func TestStatusShowsWorkspaceMode(t *testing.T) {
    // run status command, capture output
    // assert contains "Workspace Mode:"
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/golem/commands/...`
Expected: FAIL (no output)

**Step 3: Minimal implementation**

Print resolved mode:
```go
fmt.Printf("Workspace Mode: %s\n", cfg.Agents.Defaults.WorkspaceMode)
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/golem/commands/...`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/golem/commands/status.go cmd/golem/commands/status_test.go
 git commit -m "feat: show workspace mode in status" 
```

---

### Task 3: Error handling tests

**Files:**
- Modify: `internal/config/config_test.go`

**Step 1: Write failing tests**

```go
func TestWorkspacePath_PathModeRequiresWorkspace(t *testing.T) {
    cfg := DefaultConfig()
    cfg.Agents.Defaults.WorkspaceMode = "path"
    cfg.Agents.Defaults.Workspace = ""
    _, err := cfg.WorkspacePathChecked()
    if err == nil { t.Fatal("expected error") }
}
```

```go
func TestWorkspacePath_CwdModeUsesCwd(t *testing.T) {
    cfg := DefaultConfig()
    cfg.Agents.Defaults.WorkspaceMode = "cwd"
    got, err := cfg.WorkspacePathChecked()
    if err != nil { t.Fatalf("err: %v", err) }
    wd, _ := os.Getwd()
    if got != wd { t.Fatalf("got %s want %s", got, wd) }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/...`
Expected: FAIL (missing WorkspacePathChecked)

**Step 3: Minimal implementation**

- Add `WorkspacePathChecked()` returning `(string, error)`
- `WorkspacePath()` can call `WorkspacePathChecked()` and ignore error in default mode, or keep separate logic

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
 git commit -m "feat: validate workspace mode" 
```

---

### Task 4: Wire error handling to callers

**Files:**
- Modify: `internal/agent/loop.go`
- Modify: `cmd/golem/commands/status.go`
- Modify: `cmd/golem/commands/init.go`

**Step 1: Write failing test**

Add test in `cmd/golem/commands/status_test.go` that when invalid config is used (path mode, empty workspace) status returns error.

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/golem/commands/...`
Expected: FAIL

**Step 3: Minimal implementation**

- Use `WorkspacePathChecked()` in command/status/agent init and return error if invalid.
- Ensure errors are logged and program exits gracefully.

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/golem/commands/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/agent/loop.go cmd/golem/commands/status.go cmd/golem/commands/init.go cmd/golem/commands/status_test.go
 git commit -m "feat: validate workspace path usage" 
```

---

### Task 5: Docs update

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`

**Step 1: Add config docs**

Add `workspace_mode` explanation with examples for `default`, `cwd`, `path`.

**Step 2: Commit**

```bash
git add README.md README.zh-CN.md
 git commit -m "docs: document workspace mode" 
```

---

## Verification

- Run unit tests: `go test ./internal/config/...`
- Run command tests: `go test ./cmd/golem/commands/...`
- Optional: `./golem.exe status`

