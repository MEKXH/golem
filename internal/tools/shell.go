package tools

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// ExecInput parameters for exec tool
type ExecInput struct {
	Command    string `json:"command" jsonschema:"required,description=Shell command to execute"`
	WorkingDir string `json:"working_dir" jsonschema:"description=Working directory for the command"`
}

// ExecOutput result of exec tool
type ExecOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// dangerousPatterns are regex patterns that match dangerous commands.
// These are compiled once at init time for efficiency.
var dangerousPatterns = []*regexp.Regexp{
	// rm with force/recursive targeting root or home
	regexp.MustCompile(`(?i)\brm\s+(-[a-z]*r[a-z]*\s+-[a-z]*f[a-z]*|-[a-z]*f[a-z]*\s+-[a-z]*r[a-z]*|-[a-z]*rf[a-z]*|-[a-z]*fr[a-z]*)\s+/\s*$`),
	regexp.MustCompile(`(?i)\brm\s+(-[a-z]*r[a-z]*\s+-[a-z]*f[a-z]*|-[a-z]*f[a-z]*\s+-[a-z]*r[a-z]*|-[a-z]*rf[a-z]*|-[a-z]*fr[a-z]*)\s+~`),
	// sudo variants of rm
	regexp.MustCompile(`(?i)\bsudo\s+rm\s+(-[a-z]*r[a-z]*\s+-[a-z]*f[a-z]*|-[a-z]*f[a-z]*\s+-[a-z]*r[a-z]*|-[a-z]*rf[a-z]*|-[a-z]*fr[a-z]*)\s+/\s*$`),
	// explicitly disabling root safeguards
	regexp.MustCompile(`(?i)--no-preserve-root`),
	// filesystem format commands
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=`),
	// fork bomb
	regexp.MustCompile(`:\(\)\s*\{.*\|.*&\s*\}\s*;`),
	// Windows dangerous commands
	regexp.MustCompile(`(?i)\bformat\s+[a-z]:`),
	regexp.MustCompile(`(?i)\bdel\s+/[a-z]\s+/[a-z]\s+/[a-z]`),
}

// isDangerous checks whether a command matches any dangerous command pattern.
func isDangerous(cmd string) (bool, string) {
	for _, pat := range dangerousPatterns {
		if pat.MatchString(cmd) {
			return true, pat.String()
		}
	}
	return false, ""
}

type execToolImpl struct {
	timeout             time.Duration
	restrictToWorkspace bool
	workspaceDir        string
}

func (e *execToolImpl) execute(ctx context.Context, input *ExecInput) (*ExecOutput, error) {
	if dangerous, pattern := isDangerous(input.Command); dangerous {
		return &ExecOutput{
			Stderr:   fmt.Sprintf("Blocked dangerous command matching pattern: %s", pattern),
			ExitCode: 1,
		}, nil
	}

	// Determine working directory and enforce workspace restriction
	workDir := input.WorkingDir
	if e.restrictToWorkspace && e.workspaceDir != "" {
		if workDir != "" {
			if err := validatePath(workDir, e.workspaceDir); err != nil {
				return &ExecOutput{
					Stderr:   fmt.Sprintf("Working directory rejected: %s", err.Error()),
					ExitCode: 1,
				}, nil
			}
		} else {
			workDir = e.workspaceDir
		}
	} else if workDir == "" && e.workspaceDir != "" {
		workDir = e.workspaceDir
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(timeoutCtx, "cmd", "/C", input.Command)
	} else {
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", input.Command)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return &ExecOutput{
				Stderr:   err.Error(),
				ExitCode: 1,
			}, nil
		}
	}

	return &ExecOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

// NewExecTool creates the exec tool
func NewExecTool(timeoutSec int, restrictToWorkspace bool, workspaceDir string) (tool.InvokableTool, error) {
	impl := &execToolImpl{
		timeout:             time.Duration(timeoutSec) * time.Second,
		restrictToWorkspace: restrictToWorkspace,
		workspaceDir:        workspaceDir,
	}
	return utils.InferTool("exec", "Execute a shell command", impl.execute)
}
