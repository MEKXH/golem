package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/audit"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
)

func TestAuditRuntimePolicyStartup_PersistentOffWritesWarningEvent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = ""
	cfg.Policy.AllowPersistentOff = true

	loop, err := NewLoop(cfg, bus.NewMessageBus(1), nil)
	if err != nil {
		t.Fatalf("NewLoop() error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools() error: %v", err)
	}

	loop.AuditRuntimePolicyStartup(context.Background(), cfg)

	events := readAuditEvents(t, loop.workspacePath)
	if len(events) != 2 {
		t.Fatalf("expected 2 startup audit events, got %d", len(events))
	}

	if events[0].Type != "policy_startup" {
		t.Fatalf("expected first event type policy_startup, got %q", events[0].Type)
	}
	if !strings.Contains(events[0].Result, "mode=off") {
		t.Fatalf("expected startup event result to include mode=off, got %q", events[0].Result)
	}

	if events[1].Type != "policy_startup_persistent_off" {
		t.Fatalf("expected second event type policy_startup_persistent_off, got %q", events[1].Type)
	}
	if !strings.Contains(strings.ToLower(events[1].Result), "high-risk") {
		t.Fatalf("expected persistent off warning to include high-risk, got %q", events[1].Result)
	}
}

func TestAuditRuntimePolicyStartup_StrictWritesSingleEvent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := config.DefaultConfig()
	cfg.Policy.Mode = "strict"

	loop, err := NewLoop(cfg, bus.NewMessageBus(1), nil)
	if err != nil {
		t.Fatalf("NewLoop() error: %v", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		t.Fatalf("RegisterDefaultTools() error: %v", err)
	}

	loop.AuditRuntimePolicyStartup(context.Background(), cfg)

	events := readAuditEvents(t, loop.workspacePath)
	if len(events) != 1 {
		t.Fatalf("expected 1 startup audit event, got %d", len(events))
	}
	if events[0].Type != "policy_startup" {
		t.Fatalf("expected event type policy_startup, got %q", events[0].Type)
	}
	if !strings.Contains(events[0].Result, "mode=strict") {
		t.Fatalf("expected startup event result to include mode=strict, got %q", events[0].Result)
	}
}

func readAuditEvents(t *testing.T, workspacePath string) []audit.Event {
	t.Helper()

	auditPath := filepath.Join(workspacePath, "state", "audit.jsonl")
	file, err := os.Open(auditPath)
	if err != nil {
		t.Fatalf("Open audit file error: %v", err)
	}
	defer file.Close()

	events := make([]audit.Event, 0, 2)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var evt audit.Event
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			t.Fatalf("Unmarshal audit event error: %v", err)
		}
		events = append(events, evt)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Scan audit file error: %v", err)
	}
	return events
}
