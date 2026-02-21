package metrics

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRuntimeMetrics_AggregatesToolAndChannelStats(t *testing.T) {
	workspace := t.TempDir()
	recorder := NewRuntimeMetrics(workspace)
	defer recorder.Close()

	snap, err := recorder.RecordToolExecution(120*time.Millisecond, "", nil)
	if err != nil {
		t.Fatalf("RecordToolExecution success error: %v", err)
	}
	if snap.Tool.Total != 1 || snap.Tool.Errors != 0 || snap.Tool.Timeouts != 0 {
		t.Fatalf("unexpected first tool snapshot: %+v", snap.Tool)
	}

	_, _ = recorder.RecordToolExecution(250*time.Millisecond, "", errors.New("exec failed"))
	_, _ = recorder.RecordToolExecution(2*time.Second, "", context.DeadlineExceeded)
	snap, _ = recorder.RecordToolExecution(1500*time.Millisecond, "", errors.New("request timed out"))

	if snap.Tool.Total != 4 {
		t.Fatalf("expected 4 tool executions, got %d", snap.Tool.Total)
	}
	if snap.Tool.Errors != 3 {
		t.Fatalf("expected 3 tool errors, got %d", snap.Tool.Errors)
	}
	if snap.Tool.Timeouts != 2 {
		t.Fatalf("expected 2 tool timeouts, got %d", snap.Tool.Timeouts)
	}
	if got := snap.Tool.ErrorRatio(); got < 0.74 || got > 0.76 {
		t.Fatalf("expected error ratio about 0.75, got %.4f", got)
	}
	if got := snap.Tool.TimeoutRatio(); got < 0.49 || got > 0.51 {
		t.Fatalf("expected timeout ratio about 0.50, got %.4f", got)
	}
	if snap.Tool.P95ProxyLatencyMs <= 0 {
		t.Fatalf("expected p95 proxy latency > 0, got %d", snap.Tool.P95ProxyLatencyMs)
	}

	_, _ = recorder.RecordChannelSend(true)
	_, _ = recorder.RecordChannelSend(false)
	snap, _ = recorder.RecordChannelSend(true)

	if snap.Channel.SendAttempts != 3 || snap.Channel.SendFailures != 1 {
		t.Fatalf("unexpected channel snapshot: %+v", snap.Channel)
	}
	if got := snap.Channel.FailureRatio(); got < 0.33 || got > 0.34 {
		t.Fatalf("expected channel failure ratio about 0.3333, got %.4f", got)
	}
}

func TestRuntimeMetrics_ReadRuntimeSnapshot(t *testing.T) {
	workspace := t.TempDir()
	recorder := NewRuntimeMetrics(workspace)
	if _, err := recorder.RecordToolExecution(99*time.Millisecond, "", nil); err != nil {
		t.Fatalf("RecordToolExecution error: %v", err)
	}
	if _, err := recorder.RecordChannelSend(false); err != nil {
		t.Fatalf("RecordChannelSend error: %v", err)
	}

	// Must close to flush to disk
	recorder.Close()

	snap, err := ReadRuntimeSnapshot(workspace)
	if err != nil {
		t.Fatalf("ReadRuntimeSnapshot error: %v", err)
	}
	if snap.Tool.Total != 1 || snap.Channel.SendAttempts != 1 || snap.Channel.SendFailures != 1 {
		t.Fatalf("unexpected loaded snapshot: %+v", snap)
	}
}

func TestRuntimeMetrics_RecordMemoryRecall(t *testing.T) {
	workspace := t.TempDir()
	recorder := NewRuntimeMetrics(workspace)

	sourceHits := map[string]int{
		"long_term":     1,
		"diary_recent":  2,
		"diary_keyword": 1,
	}
	snap, err := recorder.RecordMemoryRecall(4, sourceHits)
	if err != nil {
		t.Fatalf("RecordMemoryRecall error: %v", err)
	}
	if snap.Memory.Recalls != 1 || snap.Memory.TotalItems != 4 || snap.Memory.LastItems != 4 {
		t.Fatalf("unexpected memory summary: %+v", snap.Memory)
	}
	if snap.Memory.LongTermHits != 1 || snap.Memory.DiaryRecentHits != 2 || snap.Memory.DiaryKeywordHits != 1 {
		t.Fatalf("unexpected memory source hits: %+v", snap.Memory)
	}

	// Must close to flush to disk
	recorder.Close()

	loaded, err := ReadRuntimeSnapshot(workspace)
	if err != nil {
		t.Fatalf("ReadRuntimeSnapshot error: %v", err)
	}
	if loaded.Memory.TotalItems != 4 {
		t.Fatalf("expected persisted memory items=4, got %+v", loaded.Memory)
	}
}
