package cron

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func tempStorePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "cron", "jobs.json")
}

func TestAddAndListJobs(t *testing.T) {
	svc := NewService(tempStorePath(t), nil)
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()

	every := int64(60000)
	job, err := svc.AddJob("test-job", "hello", Schedule{Kind: "every", EveryMS: &every}, "cli", "direct", false)
	if err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	if job.ID == "" {
		t.Fatal("expected non-empty job ID")
	}
	if job.Name != "test-job" {
		t.Fatalf("expected name 'test-job', got %q", job.Name)
	}

	jobs := svc.ListJobs(true)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
}

func TestRemoveJob(t *testing.T) {
	svc := NewService(tempStorePath(t), nil)
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()

	every := int64(60000)
	job, _ := svc.AddJob("rm-test", "msg", Schedule{Kind: "every", EveryMS: &every}, "cli", "direct", false)

	if err := svc.RemoveJob(job.ID); err != nil {
		t.Fatalf("RemoveJob: %v", err)
	}
	if len(svc.ListJobs(true)) != 0 {
		t.Fatal("expected 0 jobs after remove")
	}

	if err := svc.RemoveJob("nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent job")
	}
}

func TestEnableDisableJob(t *testing.T) {
	svc := NewService(tempStorePath(t), nil)
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()

	every := int64(60000)
	job, _ := svc.AddJob("toggle-test", "msg", Schedule{Kind: "every", EveryMS: &every}, "cli", "direct", false)

	updated, err := svc.EnableJob(job.ID, false)
	if err != nil {
		t.Fatalf("EnableJob(false): %v", err)
	}
	if updated.Enabled {
		t.Fatal("expected disabled")
	}

	enabled := svc.ListJobs(false)
	if len(enabled) != 0 {
		t.Fatal("expected 0 enabled jobs")
	}

	updated, err = svc.EnableJob(job.ID, true)
	if err != nil {
		t.Fatalf("EnableJob(true): %v", err)
	}
	if !updated.Enabled {
		t.Fatal("expected enabled")
	}
}

func TestPersistence(t *testing.T) {
	path := tempStorePath(t)

	svc1 := NewService(path, nil)
	if err := svc1.Start(); err != nil {
		t.Fatal(err)
	}
	every := int64(60000)
	svc1.AddJob("persist-test", "msg", Schedule{Kind: "every", EveryMS: &every}, "cli", "direct", false)
	svc1.Stop()

	svc2 := NewService(path, nil)
	if err := svc2.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc2.Stop()

	jobs := svc2.ListJobs(true)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 persisted job, got %d", len(jobs))
	}
	if jobs[0].Name != "persist-test" {
		t.Fatalf("expected name 'persist-test', got %q", jobs[0].Name)
	}
}

func TestEveryJobFires(t *testing.T) {
	var fired atomic.Int32

	svc := NewService(tempStorePath(t), func(job *Job) error {
		fired.Add(1)
		return nil
	})
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()

	// Create a job that should fire immediately.
	every := int64(100000) // 100s interval
	job, _ := svc.AddJob("fire-test", "msg", Schedule{Kind: "every", EveryMS: &every}, "cli", "direct", false)

	// Manually set NextRunAtMS to now to trigger immediately.
	now := time.Now().UnixMilli()
	job.State.NextRunAtMS = &now
	svc.store.Put(job)

	// Wait for the ticker to pick it up.
	deadline := time.After(5 * time.Second)
	for fired.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("job did not fire within 5 seconds")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	if fired.Load() != 1 {
		t.Fatalf("expected 1 fire, got %d", fired.Load())
	}
}

func TestAtJobDisabledAfterRun(t *testing.T) {
	var fired atomic.Int32

	path := tempStorePath(t)
	svc := NewService(path, func(job *Job) error {
		fired.Add(1)
		return nil
	})
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()

	now := time.Now().UnixMilli()
	job, _ := svc.AddJob("at-test", "msg", Schedule{Kind: "at", AtMS: &now}, "cli", "direct", false)
	_ = job

	deadline := time.After(5 * time.Second)
	for fired.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("at-job did not fire within 5 seconds")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	// The at-job with DeleteAfterRun should be deleted.
	jobs := svc.ListJobs(true)
	if len(jobs) != 0 {
		t.Fatalf("expected at-job to be deleted, got %d jobs", len(jobs))
	}
}

func TestStatus(t *testing.T) {
	svc := NewService(tempStorePath(t), nil)
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()

	status := svc.Status()
	if status["running"] != true {
		t.Fatal("expected running=true")
	}
	if status["total_jobs"] != 0 {
		t.Fatalf("expected 0 jobs, got %v", status["total_jobs"])
	}
}

func TestCronSchedule(t *testing.T) {
	svc := NewService(tempStorePath(t), nil)
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()

	job, err := svc.AddJob("cron-test", "msg", Schedule{Kind: "cron", Expr: "* * * * *"}, "cli", "direct", false)
	if err != nil {
		t.Fatalf("AddJob cron: %v", err)
	}
	if job.State.NextRunAtMS == nil {
		t.Fatal("expected NextRunAtMS to be set for cron job")
	}
}

func TestStoreLoadNonexistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "jobs.json")
	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load nonexistent should not error: %v", err)
	}
	if len(store.All()) != 0 {
		t.Fatal("expected empty store")
	}
}

func TestStoreLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jobs.json")
	os.WriteFile(path, []byte("not json"), 0644)

	store := NewStore(path)
	if err := store.Load(); err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
}
