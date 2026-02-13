package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/cron"
)

func newTestCronService(t *testing.T) *cron.Service {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cron", "jobs.json")
	svc := cron.NewService(path, nil)
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Stop)
	return svc
}

func TestCronTool_AddAndList(t *testing.T) {
	svc := newTestCronService(t)
	cronTool, err := NewCronTool(svc)
	if err != nil {
		t.Fatalf("NewCronTool: %v", err)
	}

	ctx := context.Background()

	// Add a job
	result, err := cronTool.InvokableRun(ctx, `{"action":"add","name":"test-cron","message":"hello world","every_seconds":3600}`)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(result, "Job created") {
		t.Fatalf("expected 'Job created' in result, got: %s", result)
	}

	// List jobs
	result, err = cronTool.InvokableRun(ctx, `{"action":"list"}`)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(result, "test-cron") {
		t.Fatalf("expected job name in list, got: %s", result)
	}
}

func TestCronTool_RemoveJob(t *testing.T) {
	svc := newTestCronService(t)
	cronTool, _ := NewCronTool(svc)
	ctx := context.Background()

	// Add
	cronTool.InvokableRun(ctx, `{"action":"add","name":"rm-test","message":"msg","every_seconds":60}`)

	jobs := svc.ListJobs(true)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	jobID := jobs[0].ID

	// Remove
	result, err := cronTool.InvokableRun(ctx, `{"action":"remove","job_id":"`+jobID+`"}`)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !strings.Contains(result, "removed") {
		t.Fatalf("expected 'removed' in result, got: %s", result)
	}

	if len(svc.ListJobs(true)) != 0 {
		t.Fatal("expected 0 jobs after remove")
	}
}

func TestCronTool_EnableDisable(t *testing.T) {
	svc := newTestCronService(t)
	cronTool, _ := NewCronTool(svc)
	ctx := context.Background()

	cronTool.InvokableRun(ctx, `{"action":"add","name":"toggle","message":"msg","every_seconds":60}`)
	jobID := svc.ListJobs(true)[0].ID

	// Disable
	result, err := cronTool.InvokableRun(ctx, `{"action":"disable","job_id":"`+jobID+`"}`)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if !strings.Contains(result, "disabled") {
		t.Fatalf("expected 'disabled', got: %s", result)
	}

	// Enable
	result, err = cronTool.InvokableRun(ctx, `{"action":"enable","job_id":"`+jobID+`"}`)
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !strings.Contains(result, "enabled") {
		t.Fatalf("expected 'enabled', got: %s", result)
	}
}

func TestCronTool_AddWithCronExpr(t *testing.T) {
	svc := newTestCronService(t)
	cronTool, _ := NewCronTool(svc)
	ctx := context.Background()

	result, err := cronTool.InvokableRun(ctx, `{"action":"add","name":"daily","message":"good morning","cron_expr":"0 9 * * *"}`)
	if err != nil {
		t.Fatalf("add cron: %v", err)
	}
	if !strings.Contains(result, "Job created") {
		t.Fatalf("expected 'Job created', got: %s", result)
	}
}

func TestCronTool_AddWithTimestamp(t *testing.T) {
	svc := newTestCronService(t)
	cronTool, _ := NewCronTool(svc)
	ctx := context.Background()

	result, err := cronTool.InvokableRun(ctx, `{"action":"add","name":"once","message":"reminder","at_timestamp":"2030-01-01T00:00:00Z"}`)
	if err != nil {
		t.Fatalf("add at: %v", err)
	}
	if !strings.Contains(result, "Job created") {
		t.Fatalf("expected 'Job created', got: %s", result)
	}
}

func TestCronTool_Validations(t *testing.T) {
	svc := newTestCronService(t)
	cronTool, _ := NewCronTool(svc)
	ctx := context.Background()

	tests := []struct {
		name string
		args string
	}{
		{"missing action", `{}`},
		{"bad action", `{"action":"fly"}`},
		{"add missing name", `{"action":"add","message":"hi","every_seconds":60}`},
		{"add missing message", `{"action":"add","name":"test","every_seconds":60}`},
		{"add no schedule", `{"action":"add","name":"test","message":"hi"}`},
		{"remove missing id", `{"action":"remove"}`},
		{"enable missing id", `{"action":"enable"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cronTool.InvokableRun(ctx, tt.args)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
