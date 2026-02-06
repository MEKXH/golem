package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecTool_UsesWorkspaceDirWhenWorkingDirEmpty(t *testing.T) {
    tmpDir := t.TempDir()
    tool, err := NewExecTool(60, false, tmpDir)
    if err != nil {
        t.Fatalf("NewExecTool error: %v", err)
    }

    cmd := "pwd"
    if runtime.GOOS == "windows" {
        cmd = "cd"
    }

    ctx := context.Background()
    argsJSON := fmt.Sprintf(`{"command": %q}`, cmd)

    result, err := tool.InvokableRun(ctx, argsJSON)
    if err != nil {
        t.Fatalf("InvokableRun error: %v", err)
    }

    stdout := result
    var out ExecOutput
    if err := json.Unmarshal([]byte(result), &out); err == nil {
        stdout = out.Stdout
    }

    if !strings.Contains(stdout, tmpDir) {
        if runtime.GOOS == "windows" {
            escaped := strings.ReplaceAll(tmpDir, "\\", "\\\\")
            if strings.Contains(stdout, escaped) {
                return
            }
        }
        t.Fatalf("expected command to run in workspace dir %q, got output: %s", tmpDir, stdout)
    }
}

func TestExecTool_DangerousCommands(t *testing.T) {
	dangerousCmds := []struct {
		name    string
		command string
	}{
		{"rm -rf /", "rm -rf /"},
		{"rm -r -f /", "rm -r -f /"},
		{"rm -fr /", "rm -fr /"},
		{"sudo rm -rf /", "sudo rm -rf /"},
		{"rm -rf ~", "rm -rf ~"},
		{"mkfs.ext4 /dev/sda", "mkfs.ext4 /dev/sda"},
		{"dd if=/dev/zero of=/dev/sda", "dd if=/dev/zero of=/dev/sda"},
		{"fork bomb", ":(){:|:&};:"},
	}

	for _, tc := range dangerousCmds {
		t.Run(tc.name, func(t *testing.T) {
			tool, err := NewExecTool(60, false, "")
			if err != nil {
				t.Fatalf("NewExecTool error: %v", err)
			}

			ctx := context.Background()
			argsJSON := fmt.Sprintf(`{"command": %q}`, tc.command)

			result, err := tool.InvokableRun(ctx, argsJSON)
			if err != nil {
				t.Fatalf("InvokableRun error: %v", err)
			}

			var out ExecOutput
			if err := json.Unmarshal([]byte(result), &out); err != nil {
				t.Fatalf("failed to unmarshal result: %v, raw: %s", err, result)
			}

			if out.ExitCode == 0 {
				t.Errorf("expected non-zero exit code for dangerous command %q, got 0", tc.command)
			}
			if !strings.Contains(out.Stderr, "Blocked") {
				t.Errorf("expected stderr to contain 'Blocked' for command %q, got: %s", tc.command, out.Stderr)
			}
		})
	}
}

func TestExecTool_SafeCommandsAllowed(t *testing.T) {
	tool, err := NewExecTool(60, false, "")
	if err != nil {
		t.Fatalf("NewExecTool error: %v", err)
	}

	ctx := context.Background()
	argsJSON := `{"command": "echo hello"}`

	result, err := tool.InvokableRun(ctx, argsJSON)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}

	var out ExecOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		// If not JSON, check raw output
		if !strings.Contains(result, "hello") {
			t.Errorf("expected output to contain 'hello', got: %s", result)
		}
		return
	}

	if out.ExitCode != 0 {
		t.Errorf("expected exit code 0 for safe command, got %d (stderr: %s)", out.ExitCode, out.Stderr)
	}
	if !strings.Contains(out.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got: %s", out.Stdout)
	}
}

func TestExecTool_RestrictToWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(60, true, tmpDir)
	if err != nil {
		t.Fatalf("NewExecTool error: %v", err)
	}

	// Try to run a command with a working directory outside the workspace
	outsideDir := filepath.Dir(tmpDir) // parent of tmpDir is outside workspace
	ctx := context.Background()
	argsJSON := fmt.Sprintf(`{"command": "echo test", "working_dir": %q}`, outsideDir)

	result, err := tool.InvokableRun(ctx, argsJSON)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}

	var out ExecOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("failed to unmarshal result: %v, raw: %s", err, result)
	}

	if out.ExitCode == 0 {
		t.Error("expected non-zero exit code when working dir is outside workspace")
	}
	if !strings.Contains(out.Stderr, "rejected") && !strings.Contains(out.Stderr, "denied") && !strings.Contains(out.Stderr, "outside") {
		t.Errorf("expected rejection message in stderr, got: %s", out.Stderr)
	}
}
