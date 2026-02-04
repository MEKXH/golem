package commands

import (
    "strings"
    "testing"
)

func TestStatusCommand_PrintsConfig(t *testing.T) {
    tmpDir := t.TempDir()
    t.Setenv("HOME", tmpDir)
    t.Setenv("USERPROFILE", tmpDir)

    output := captureOutput(t, func() {
        if err := runStatus(nil, nil); err != nil {
            t.Fatalf("runStatus error: %v", err)
        }
    })

    if !strings.Contains(output, "Golem Status") {
        t.Fatalf("expected status output, got: %s", output)
    }
    if !strings.Contains(output, "Config:") {
        t.Fatalf("expected config line, got: %s", output)
    }
    if !strings.Contains(output, "Mode: default") {
        t.Fatalf("expected workspace mode line, got: %s", output)
    }
}
