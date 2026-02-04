package commands

import (
    "strings"
    "testing"
)

func TestChatCommand_SingleMessageNoProvider(t *testing.T) {
    tmpDir := t.TempDir()
    t.Setenv("HOME", tmpDir)
    t.Setenv("USERPROFILE", tmpDir)

    output := captureOutput(t, func() {
        if err := runChat(nil, []string{"hello"}); err != nil {
            t.Fatalf("runChat error: %v", err)
        }
    })

    if !strings.Contains(output, "No model configured") {
        t.Fatalf("expected output to mention 'No model configured', got: %s", output)
    }
}
