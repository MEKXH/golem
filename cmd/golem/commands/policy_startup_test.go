package commands

import "testing"

func TestPersistentOffWarningMessage_OnlyWhenOffWithoutTTL(t *testing.T) {
	warnMsg, shouldWarn := persistentOffWarningMessage("off", "")
	if !shouldWarn {
		t.Fatal("expected warning for policy.mode=off without off_ttl")
	}
	if warnMsg == "" {
		t.Fatal("expected non-empty warning message")
	}

	if msg, ok := persistentOffWarningMessage("off", "10m"); ok || msg != "" {
		t.Fatalf("expected no warning when off_ttl is set, got ok=%t msg=%q", ok, msg)
	}

	if msg, ok := persistentOffWarningMessage("strict", ""); ok || msg != "" {
		t.Fatalf("expected no warning in strict mode, got ok=%t msg=%q", ok, msg)
	}
}
