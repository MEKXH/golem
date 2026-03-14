package geocodebook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderListPatterns_RanksIntentMatches(t *testing.T) {
	workspace := t.TempDir()
	writeTestCodebook(t, workspace, `name: postgis-core
description: Common PostGIS query patterns
patterns:
  - name: point-buffer-count
    description: Count features within a buffer around a point
    tags: [buffer, count, point]
    template: |
      SELECT COUNT(*) FROM {{target_table}}
      WHERE ST_DWithin(
        {{geom_col}}::geography,
        ST_SetSRID(ST_MakePoint({{lon}}, {{lat}}), 4326)::geography,
        {{distance_m}}
      )
    variables:
      target_table: { type: table, required: true }
      geom_col: { type: column, default: geom }
      lon: { type: float, required: true }
      lat: { type: float, required: true }
      distance_m: { type: float, required: true }
    verified: true
    success_rate: 0.98
  - name: nearest-neighbor
    description: Find the nearest geometry to a point
    tags: [nearest, distance]
    template: |
      SELECT * FROM {{target_table}}
      ORDER BY {{geom_col}} <-> ST_SetSRID(ST_MakePoint({{lon}}, {{lat}}), 4326)
      LIMIT 1
    variables:
      target_table: { type: table, required: true }
      geom_col: { type: column, default: geom }
      lon: { type: float, required: true }
      lat: { type: float, required: true }
    verified: true
    success_rate: 0.91
`)

	loader := NewLoader(workspace)
	matches, err := loader.ListPatterns("count schools within 500m buffer around a point", 5)
	if err != nil {
		t.Fatalf("ListPatterns() error = %v", err)
	}
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
	if matches[0].Name != "point-buffer-count" {
		t.Fatalf("expected strongest match point-buffer-count, got %+v", matches[0])
	}
}

func TestLoaderRenderPattern_AppliesDefaultsAndVariables(t *testing.T) {
	workspace := t.TempDir()
	writeTestCodebook(t, workspace, `name: postgis-core
description: Common PostGIS query patterns
patterns:
  - name: point-buffer-count
    description: Count features within a buffer around a point
    template: |
      SELECT COUNT(*) FROM {{target_table}}
      WHERE ST_DWithin(
        {{geom_col}}::geography,
        ST_SetSRID(ST_MakePoint({{lon}}, {{lat}}), 4326)::geography,
        {{distance_m}}
      )
    variables:
      target_table: { type: table, required: true }
      geom_col: { type: column, default: geom }
      lon: { type: float, required: true }
      lat: { type: float, required: true }
      distance_m: { type: float, required: true }
    verified: true
    success_rate: 0.98
`)

	loader := NewLoader(workspace)
	rendered, err := loader.RenderPattern("point-buffer-count", map[string]string{
		"target_table": "schools",
		"lon":          "116.397",
		"lat":          "39.908",
		"distance_m":   "500",
	})
	if err != nil {
		t.Fatalf("RenderPattern() error = %v", err)
	}
	if !strings.Contains(rendered.SQL, "FROM schools") {
		t.Fatalf("expected target_table substitution, got %q", rendered.SQL)
	}
	if !strings.Contains(rendered.SQL, "geom::geography") {
		t.Fatalf("expected default geom_col substitution, got %q", rendered.SQL)
	}
}

func TestLoaderRenderPattern_RequiresMissingVariables(t *testing.T) {
	workspace := t.TempDir()
	writeTestCodebook(t, workspace, `name: postgis-core
description: Common PostGIS query patterns
patterns:
  - name: point-buffer-count
    description: Count features within a buffer around a point
    template: |
      SELECT COUNT(*) FROM {{target_table}} WHERE {{geom_col}} IS NOT NULL
    variables:
      target_table: { type: table, required: true }
      geom_col: { type: column, required: true }
    verified: true
`)

	loader := NewLoader(workspace)
	_, err := loader.RenderPattern("point-buffer-count", map[string]string{
		"target_table": "schools",
	})
	if err == nil {
		t.Fatal("expected missing required variable error")
	}
}

func TestLoaderBuildSummary_IncludesPatternNames(t *testing.T) {
	workspace := t.TempDir()
	writeTestCodebook(t, workspace, `name: postgis-core
description: Common PostGIS query patterns
patterns:
  - name: point-buffer-count
    description: Count features within a buffer around a point
    tags: [buffer, count]
    template: SELECT 1
    verified: true
    success_rate: 0.98
`)

	summary, err := NewLoader(workspace).BuildSummary()
	if err != nil {
		t.Fatalf("BuildSummary() error = %v", err)
	}
	if !strings.Contains(summary, "Spatial SQL Codebook") || !strings.Contains(summary, "point-buffer-count") {
		t.Fatalf("expected summary to mention codebook pattern, got %q", summary)
	}
}

func writeTestCodebook(t *testing.T, workspace, content string) {
	t.Helper()
	dir := filepath.Join(workspace, "geo-codebook")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "patterns.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
