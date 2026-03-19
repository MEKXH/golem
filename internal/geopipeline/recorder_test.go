package geopipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecorderSave_WritesPipelineRecord(t *testing.T) {
	workspace := t.TempDir()
	recorder := NewRecorder(workspace)
	recorder.now = func() string { return "20260314-213000" }

	err := recorder.Save("analyze river sinuosity", []Step{
		{Tool: "geo_info", ArgsJSON: `{"path":"river.geojson"}`},
		{Tool: "geo_sinuosity", ArgsJSON: `{"input_path":"river.geojson"}`},
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(workspace, "pipelines", "geo", "*.yaml"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 learned pipeline file, got %d", len(matches))
	}

	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "analyze river sinuosity") {
		t.Fatalf("expected goal in saved pipeline, got %s", text)
	}
	if !strings.Contains(text, "geo_sinuosity") {
		t.Fatalf("expected tool sequence in saved pipeline, got %s", text)
	}
}

func TestRecorderBuildSummary_IncludesLearnedPipelines(t *testing.T) {
	workspace := t.TempDir()
	dir := filepath.Join(workspace, "pipelines", "geo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := `id: pipeline-1
goal: analyze river sinuosity
created_at: "2026-03-14T21:30:00Z"
steps:
  - tool: geo_info
    args_json: '{"path":"river.geojson"}'
  - tool: geo_sinuosity
    args_json: '{"input_path":"river.geojson"}'
`
	if err := os.WriteFile(filepath.Join(dir, "pipeline-1.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	summary := NewRecorder(workspace).BuildSummary()
	if !strings.Contains(summary, "Learned Geo Pipelines") {
		t.Fatalf("expected learned pipelines section, got %s", summary)
	}
	if !strings.Contains(summary, "analyze river sinuosity") || !strings.Contains(summary, "geo_sinuosity") {
		t.Fatalf("expected learned pipeline content in summary, got %s", summary)
	}
}
