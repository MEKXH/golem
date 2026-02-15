package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadWriteLongTermMemory(t *testing.T) {
	mgr := NewManager(t.TempDir())
	if err := mgr.WriteLongTerm("remember this"); err != nil {
		t.Fatalf("WriteLongTerm error: %v", err)
	}

	got, err := mgr.ReadLongTerm()
	if err != nil {
		t.Fatalf("ReadLongTerm error: %v", err)
	}
	if got != "remember this" {
		t.Fatalf("expected memory content, got %q", got)
	}
}

func TestAppendDiaryAt(t *testing.T) {
	mgr := NewManager(t.TempDir())
	now := time.Date(2026, 2, 11, 10, 30, 0, 0, time.UTC)

	diaryPath, err := mgr.AppendDiaryAt(now, "met with product team")
	if err != nil {
		t.Fatalf("AppendDiaryAt error: %v", err)
	}
	if !strings.HasSuffix(diaryPath, filepath.Join("memory", "2026-02-11.md")) {
		t.Fatalf("unexpected diary path: %s", diaryPath)
	}

	content, err := mgr.ReadDiary("2026-02-11")
	if err != nil {
		t.Fatalf("ReadDiary error: %v", err)
	}
	if !strings.Contains(content, "met with product team") {
		t.Fatalf("expected diary content, got: %s", content)
	}
}

func TestReadRecentDiaries(t *testing.T) {
	mgr := NewManager(t.TempDir())
	dates := []string{"2026-02-08", "2026-02-09", "2026-02-10", "2026-02-11"}
	for i, d := range dates {
		ts, _ := time.Parse("2006-01-02", d)
		if _, err := mgr.AppendDiaryAt(ts, "entry-"+d); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	entries, err := mgr.ReadRecentDiaries(3)
	if err != nil {
		t.Fatalf("ReadRecentDiaries error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Date != "2026-02-09" || entries[2].Date != "2026-02-11" {
		t.Fatalf("unexpected dates order: %#v", entries)
	}
}

func TestRecallContext_RecentAndKeywordSources(t *testing.T) {
	workspace := t.TempDir()
	mgr := NewManager(workspace)

	if err := mgr.WriteLongTerm("payment service timeout mitigation runbook"); err != nil {
		t.Fatalf("WriteLongTerm: %v", err)
	}

	_ = os.WriteFile(filepath.Join(workspace, "memory", "2026-02-12.md"), []byte("- [10:00:00] shipping update"), 0o644)
	_ = os.WriteFile(filepath.Join(workspace, "memory", "2026-02-13.md"), []byte("- [10:00:00] payment timeout investigation"), 0o644)
	_ = os.WriteFile(filepath.Join(workspace, "memory", "2026-02-14.md"), []byte("- [10:00:00] api incident wrap-up"), 0o644)
	_ = os.WriteFile(filepath.Join(workspace, "memory", "2026-02-15.md"), []byte("- [10:00:00] deployment status"), 0o644)

	recall, err := mgr.RecallContext("investigate payment timeout", 3, 3)
	if err != nil {
		t.Fatalf("RecallContext: %v", err)
	}
	if recall.RecallCount == 0 {
		t.Fatalf("expected non-empty recall result: %+v", recall)
	}
	if recall.SourceHits["long_term"] == 0 {
		t.Fatalf("expected long_term hit in source stats: %+v", recall.SourceHits)
	}
	if recall.SourceHits["diary_recent"] == 0 {
		t.Fatalf("expected diary_recent hit in source stats: %+v", recall.SourceHits)
	}
	if recall.SourceHits["diary_keyword"] == 0 {
		t.Fatalf("expected diary_keyword hit in source stats: %+v", recall.SourceHits)
	}
}

func TestRecallContext_NoKeywordDoesNotInjectLongTerm(t *testing.T) {
	workspace := t.TempDir()
	mgr := NewManager(workspace)

	if err := mgr.WriteLongTerm("kernel panic troubleshooting notes"); err != nil {
		t.Fatalf("WriteLongTerm: %v", err)
	}
	if _, err := mgr.AppendDiaryAt(time.Date(2026, 2, 15, 8, 0, 0, 0, time.UTC), "daily summary"); err != nil {
		t.Fatalf("AppendDiaryAt: %v", err)
	}

	recall, err := mgr.RecallContext("prepare weekly sync agenda", 2, 2)
	if err != nil {
		t.Fatalf("RecallContext: %v", err)
	}
	if recall.SourceHits["long_term"] != 0 {
		t.Fatalf("expected no long_term hit for unrelated query, got %+v", recall.SourceHits)
	}
}
