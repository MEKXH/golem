package skills

import "testing"

func TestBuildTelemetryReport_ListsCountsAndSuccessRatio(t *testing.T) {
	report := BuildTelemetryReport(TelemetrySnapshot{
		Skills: map[string]TelemetryEntry{
			"spatial-analysis": {Shown: 4, Selected: 2, Success: 1, Failure: 1},
		},
	})
	if len(report.Entries) != 1 {
		t.Fatalf("expected one report entry, got %+v", report.Entries)
	}
	entry := report.Entries[0]
	if entry.Name != "spatial-analysis" {
		t.Fatalf("unexpected report entry %+v", entry)
	}
	if entry.Shown != 4 || entry.Selected != 2 || entry.Success != 1 || entry.Failure != 1 {
		t.Fatalf("expected counters to be preserved, got %+v", entry)
	}
	if entry.SuccessRatio != 0.5 {
		t.Fatalf("expected success ratio 0.5, got %v", entry.SuccessRatio)
	}
	if entry.HasOutcomeData != true {
		t.Fatalf("expected report entry to note outcome data, got %+v", entry)
	}
}

func TestBuildTelemetryReport_SortsLowestPerformingSkillsFirstWhenOutcomeDataExists(t *testing.T) {
	report := BuildTelemetryReport(TelemetrySnapshot{
		Skills: map[string]TelemetryEntry{
			"remote-sensing":   {Shown: 5, Selected: 3, Success: 0, Failure: 3},
			"spatial-analysis": {Shown: 5, Selected: 3, Success: 2, Failure: 1},
			"data-pipeline":    {Shown: 5, Selected: 1, Success: 0, Failure: 0},
		},
	})
	if len(report.Entries) != 3 {
		t.Fatalf("expected three report entries, got %+v", report.Entries)
	}
	if report.Entries[0].Name != "remote-sensing" {
		t.Fatalf("expected lowest-performing skill first, got %+v", report.Entries)
	}
	if report.Entries[1].Name != "spatial-analysis" {
		t.Fatalf("expected higher-performing skill second, got %+v", report.Entries)
	}
	if report.Entries[2].Name != "data-pipeline" {
		t.Fatalf("expected skill without outcome data last, got %+v", report.Entries)
	}
}
