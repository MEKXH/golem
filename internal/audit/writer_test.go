package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWriter_AppendEvent(t *testing.T) {
	workspace := t.TempDir()
	writer := NewWriter(workspace)

	firstTime := time.Date(2026, 2, 15, 8, 0, 0, 0, time.UTC)
	secondTime := firstTime.Add(5 * time.Second)

	if err := writer.Append(Event{
		Time:      firstTime,
		Type:      "policy_decision",
		RequestID: "req-1",
		Tool:      "exec",
		Result:    "allow",
	}); err != nil {
		t.Fatalf("Append first event error: %v", err)
	}

	if err := writer.Append(Event{
		Time:      secondTime,
		Type:      "tool_result",
		RequestID: "req-1",
		Tool:      "exec",
		Result:    "ok",
	}); err != nil {
		t.Fatalf("Append second event error: %v", err)
	}

	auditPath := filepath.Join(workspace, "state", "audit.jsonl")
	file, err := os.Open(auditPath)
	if err != nil {
		t.Fatalf("Open audit file error: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0, 2)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan audit file error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 jsonl lines, got %d", len(lines))
	}

	var first Event
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first line error: %v", err)
	}
	if !first.Time.Equal(firstTime) {
		t.Fatalf("expected first time %s, got %s", firstTime, first.Time)
	}
	if first.Type != "policy_decision" {
		t.Fatalf("expected first type policy_decision, got %q", first.Type)
	}
	if first.RequestID != "req-1" {
		t.Fatalf("expected first request_id req-1, got %q", first.RequestID)
	}
	if first.Tool != "exec" {
		t.Fatalf("expected first tool exec, got %q", first.Tool)
	}
	if first.Result != "allow" {
		t.Fatalf("expected first result allow, got %q", first.Result)
	}

	var second Event
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("unmarshal second line error: %v", err)
	}
	if !second.Time.Equal(secondTime) {
		t.Fatalf("expected second time %s, got %s", secondTime, second.Time)
	}
	if second.Type != "tool_result" {
		t.Fatalf("expected second type tool_result, got %q", second.Type)
	}
	if second.RequestID != "req-1" {
		t.Fatalf("expected second request_id req-1, got %q", second.RequestID)
	}
	if second.Tool != "exec" {
		t.Fatalf("expected second tool exec, got %q", second.Tool)
	}
	if second.Result != "ok" {
		t.Fatalf("expected second result ok, got %q", second.Result)
	}
}

func TestWriter_AppendEvent_MkdirAllFailure(t *testing.T) {
	workspace := t.TempDir()
	statePath := filepath.Join(workspace, "state")
	if err := os.WriteFile(statePath, []byte("not-a-dir"), 0644); err != nil {
		t.Fatalf("WriteFile state blocker error: %v", err)
	}

	writer := NewWriter(workspace)
	err := writer.Append(Event{Time: time.Now().UTC(), Type: "policy_decision"})
	if err == nil {
		t.Fatal("expected append error when state path is a file")
	}
}

func TestWriter_AppendEvent_Concurrent(t *testing.T) {
	workspace := t.TempDir()
	writer := NewWriter(workspace)

	const total = 20
	var wg sync.WaitGroup
	errCh := make(chan error, total)
	wg.Add(total)
	for i := 0; i < total; i++ {
		i := i
		go func() {
			defer wg.Done()
			if err := writer.Append(Event{
				Time:      time.Date(2026, 2, 15, 9, 0, i, 0, time.UTC),
				Type:      "tool_result",
				RequestID: fmt.Sprintf("req-%d", i),
				Tool:      "exec",
				Result:    "ok",
			}); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("append failed in concurrent path: %v", err)
	}

	auditPath := filepath.Join(workspace, "state", "audit.jsonl")
	file, err := os.Open(auditPath)
	if err != nil {
		t.Fatalf("Open audit file error: %v", err)
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan audit file error: %v", err)
	}
	if count != total {
		t.Fatalf("expected %d lines, got %d", total, count)
	}
}
