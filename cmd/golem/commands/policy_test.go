package commands

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/audit"
	"github.com/MEKXH/golem/internal/config"
)

func TestPolicyOff_RequiresTTL(t *testing.T) {
	preparePolicyWorkspace(t)

	cmd := newPolicyOffCmd()
	if err := runPolicyOff(cmd, nil); err == nil {
		t.Fatal("expected error when --ttl is missing")
	}
}

func TestPolicyOff_SetsConfigAndWritesAudit(t *testing.T) {
	workspacePath := preparePolicyWorkspace(t)

	cmd := newPolicyOffCmd()
	if err := cmd.Flags().Set("ttl", "30m"); err != nil {
		t.Fatalf("set --ttl: %v", err)
	}

	output := captureOutput(t, func() {
		if err := runPolicyOff(cmd, nil); err != nil {
			t.Fatalf("runPolicyOff: %v", err)
		}
	})
	if !strings.Contains(output, "Policy set to off") {
		t.Fatalf("expected success output, got: %s", output)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.Policy.Mode != "off" {
		t.Fatalf("expected policy mode off, got %q", cfg.Policy.Mode)
	}
	if cfg.Policy.OffTTL != "30m0s" {
		t.Fatalf("expected normalized off_ttl 30m0s, got %q", cfg.Policy.OffTTL)
	}
	if cfg.Policy.AllowPersistentOff {
		t.Fatal("expected allow_persistent_off=false for ttl-limited off mode")
	}

	events := readPolicyAuditEvents(t, workspacePath)
	if len(events) == 0 {
		t.Fatal("expected policy audit events")
	}
	last := events[len(events)-1]
	if last.Type != "policy_cli_switch" {
		t.Fatalf("expected event type policy_cli_switch, got %q", last.Type)
	}
	if !strings.Contains(last.Result, "mode=off") {
		t.Fatalf("expected audit event result to contain mode=off, got %q", last.Result)
	}
}

func TestPolicyStrict_ClearsOffTTL(t *testing.T) {
	preparePolicyWorkspace(t)

	offCmd := newPolicyOffCmd()
	_ = offCmd.Flags().Set("ttl", "15m")
	if err := runPolicyOff(offCmd, nil); err != nil {
		t.Fatalf("runPolicyOff: %v", err)
	}

	if err := runPolicyStrict(nil, nil); err != nil {
		t.Fatalf("runPolicyStrict: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.Policy.Mode != "strict" {
		t.Fatalf("expected strict mode, got %q", cfg.Policy.Mode)
	}
	if cfg.Policy.OffTTL != "" {
		t.Fatalf("expected off_ttl cleared, got %q", cfg.Policy.OffTTL)
	}
}

func TestPolicyRelaxed_ClearsOffTTL(t *testing.T) {
	preparePolicyWorkspace(t)

	offCmd := newPolicyOffCmd()
	_ = offCmd.Flags().Set("ttl", "10m")
	if err := runPolicyOff(offCmd, nil); err != nil {
		t.Fatalf("runPolicyOff: %v", err)
	}

	if err := runPolicyRelaxed(nil, nil); err != nil {
		t.Fatalf("runPolicyRelaxed: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.Policy.Mode != "relaxed" {
		t.Fatalf("expected relaxed mode, got %q", cfg.Policy.Mode)
	}
	if cfg.Policy.OffTTL != "" {
		t.Fatalf("expected off_ttl cleared, got %q", cfg.Policy.OffTTL)
	}
}

func TestPolicyStatus_ShowsRiskForPersistentOff(t *testing.T) {
	preparePolicyWorkspace(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = ""
	cfg.Policy.AllowPersistentOff = true
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	output := captureOutput(t, func() {
		if err := runPolicyStatus(nil, nil); err != nil {
			t.Fatalf("runPolicyStatus: %v", err)
		}
	})
	if !strings.Contains(output, "HIGH-RISK") {
		t.Fatalf("expected high-risk warning in output, got: %s", output)
	}
}

func TestPolicyCommand_RegisteredInRoot(t *testing.T) {
	root := NewRootCmd()
	found, _, err := root.Find([]string{"policy", "status"})
	if err != nil {
		t.Fatalf("find policy status command: %v", err)
	}
	if found == nil || found.Name() != "status" {
		t.Fatalf("expected status command, got %#v", found)
	}
}

func preparePolicyWorkspace(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		t.Fatalf("WorkspacePathChecked: %v", err)
	}
	return workspacePath
}

func readPolicyAuditEvents(t *testing.T, workspacePath string) []audit.Event {
	t.Helper()

	auditPath := filepath.Join(workspacePath, "state", "audit.jsonl")
	file, err := os.Open(auditPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("open audit file: %v", err)
	}
	defer file.Close()

	events := make([]audit.Event, 0, 8)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var evt audit.Event
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			t.Fatalf("unmarshal audit event: %v", err)
		}
		events = append(events, evt)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan audit file: %v", err)
	}
	return events
}
