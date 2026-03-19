package geopipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatcherFind_ReturnsRelevantPipelinesInScoreOrder(t *testing.T) {
	workspace := t.TempDir()
	dir := filepath.Join(workspace, "pipelines", "geo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	files := map[string]string{
		"pipeline-land-change.yaml": `id: pipeline-land-change
goal: analyze land use change
created_at: "2026-03-14T21:30:00Z"
steps:
  - tool: geo_data_catalog
  - tool: geo_process
`,
		"pipeline-land-change-buildings.yaml": `id: pipeline-land-change-buildings
goal: analyze land use change with building extraction
created_at: "2026-03-15T21:30:00Z"
steps:
  - tool: geo_data_catalog
  - tool: geo_process
  - tool: geo_sinuosity
`,
		"pipeline-river.yaml": `id: pipeline-river
goal: analyze river sinuosity
created_at: "2026-03-13T21:30:00Z"
steps:
  - tool: geo_info
  - tool: geo_sinuosity
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	matches, err := NewMatcher(workspace).Find("analyze land use change in another area", 3)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 relevant matches, got %+v", matches)
	}
	if matches[0].Goal != "analyze land use change" {
		t.Fatalf("expected best match to be land use change, got %+v", matches[0])
	}
	if matches[1].Goal != "analyze land use change with building extraction" {
		t.Fatalf("expected second match to be building extraction variant, got %+v", matches[1])
	}
}

func TestMatcherFind_RespectsLimitAndSkipsUnrelatedPipelines(t *testing.T) {
	workspace := t.TempDir()
	dir := filepath.Join(workspace, "pipelines", "geo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	files := map[string]string{
		"pipeline-parks.yaml": `id: pipeline-parks
goal: summarize park accessibility
created_at: "2026-03-14T21:30:00Z"
steps:
  - tool: geo_spatial_query
`,
		"pipeline-raster.yaml": `id: pipeline-raster
goal: preprocess sentinel raster tiles
created_at: "2026-03-14T21:35:00Z"
steps:
  - tool: geo_process
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	matches, err := NewMatcher(workspace).Find("analyze river sinuosity", 1)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no unrelated matches, got %+v", matches)
	}
}
