package memory

import (
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
