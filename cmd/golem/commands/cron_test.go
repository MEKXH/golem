package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/cron"
)

func TestCronRun_ExecutesJobNow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	// Ensure workspace/config exists.
	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		t.Fatalf("workspace path: %v", err)
	}

	cronStorePath := filepath.Join(workspacePath, "cron", "jobs.json")
	svc := cron.NewService(cronStorePath, nil)
	if err := svc.Start(); err != nil {
		t.Fatalf("cron start: %v", err)
	}
	everyMS := int64(60000)
	job, err := svc.AddJob("manual-run", "say hello", cron.Schedule{
		Kind:    "every",
		EveryMS: &everyMS,
	}, "", "", false)
	if err != nil {
		svc.Stop()
		t.Fatalf("add job: %v", err)
	}
	svc.Stop()

	out := captureOutput(t, func() {
		if err := runCronNow(nil, []string{job.ID}); err != nil {
			t.Fatalf("runCronNow: %v", err)
		}
	})
	if !strings.Contains(out, "executed") {
		t.Fatalf("expected executed output, got: %s", out)
	}
}
