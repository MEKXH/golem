package skills

import (
	"path/filepath"
	"testing"
)

func TestTelemetryRecorder_PersistsShownSelectedAndOutcomeCounters(t *testing.T) {
	workspace := t.TempDir()
	recorder := NewTelemetryRecorder(workspace)

	if err := recorder.RecordShown([]SkillInfo{{Name: "spatial-analysis"}, {Name: "remote-sensing"}}); err != nil {
		t.Fatalf("RecordShown() error = %v", err)
	}
	if err := recorder.RecordSelected("spatial-analysis"); err != nil {
		t.Fatalf("RecordSelected() error = %v", err)
	}
	if err := recorder.RecordOutcome("spatial-analysis", true); err != nil {
		t.Fatalf("RecordOutcome(success) error = %v", err)
	}
	if err := recorder.RecordOutcome("remote-sensing", false); err != nil {
		t.Fatalf("RecordOutcome(failure) error = %v", err)
	}

	snapshot, err := recorder.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if snapshot.Skills["spatial-analysis"].Shown != 1 {
		t.Fatalf("expected shown=1, got %+v", snapshot.Skills["spatial-analysis"])
	}
	if snapshot.Skills["spatial-analysis"].Selected != 1 {
		t.Fatalf("expected selected=1, got %+v", snapshot.Skills["spatial-analysis"])
	}
	if snapshot.Skills["spatial-analysis"].Success != 1 {
		t.Fatalf("expected success=1, got %+v", snapshot.Skills["spatial-analysis"])
	}
	if snapshot.Skills["remote-sensing"].Failure != 1 {
		t.Fatalf("expected failure=1, got %+v", snapshot.Skills["remote-sensing"])
	}
	if snapshot.Path != filepath.Join(workspace, "state", "skill_telemetry.json") {
		t.Fatalf("unexpected snapshot path %q", snapshot.Path)
	}
}

func TestTelemetryRecorder_RecordShownDeduplicatesWithinOneCall(t *testing.T) {
	workspace := t.TempDir()
	recorder := NewTelemetryRecorder(workspace)

	if err := recorder.RecordShown([]SkillInfo{{Name: "spatial-analysis"}, {Name: "spatial-analysis"}}); err != nil {
		t.Fatalf("RecordShown() error = %v", err)
	}

	snapshot, err := recorder.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if snapshot.Skills["spatial-analysis"].Shown != 1 {
		t.Fatalf("expected deduplicated shown count, got %+v", snapshot.Skills["spatial-analysis"])
	}
}

func TestSelectSkillsForQuery_MatchesExplicitWorkspaceSkillReference(t *testing.T) {
	selected := SelectSkillsForQuery([]SkillInfo{
		{Name: "spatial-analysis", Description: "workspace geo skill", Source: "workspace"},
		{Name: "weather", Description: "builtin skill", Source: "builtin"},
	}, "Use spatial analysis to inspect this raster")
	if len(selected) != 1 {
		t.Fatalf("expected exactly one selected skill, got %+v", selected)
	}
	if selected[0].Name != "spatial-analysis" {
		t.Fatalf("expected spatial-analysis to be selected, got %+v", selected[0])
	}
}
