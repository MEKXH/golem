package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGeoSQLCodebookTool_ListAndRender(t *testing.T) {
	workspace := t.TempDir()
	dir := filepath.Join(workspace, "geo-codebook")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := `name: postgis-core
description: Common PostGIS query patterns
patterns:
  - name: point-buffer-count
    description: Count features within a buffer around a point
    tags: [buffer, count, point]
    template: |
      SELECT COUNT(*) FROM {{target_table}}
      WHERE ST_DWithin({{geom_col}}::geography, ST_SetSRID(ST_MakePoint({{lon}}, {{lat}}), 4326)::geography, {{distance_m}})
    variables:
      target_table: { type: table, required: true }
      geom_col: { type: column, default: geom }
      lon: { type: float, required: true }
      lat: { type: float, required: true }
      distance_m: { type: float, required: true }
    verified: true
    success_rate: 0.98
`
	if err := os.WriteFile(filepath.Join(dir, "patterns.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tool, err := NewGeoSQLCodebookTool(workspace)
	if err != nil {
		t.Fatalf("NewGeoSQLCodebookTool() error = %v", err)
	}

	listRaw, err := tool.InvokableRun(context.Background(), `{"action":"list","intent":"count schools in buffer"}`)
	if err != nil {
		t.Fatalf("list InvokableRun() error = %v", err)
	}
	var listOut GeoSQLCodebookOutput
	if err := json.Unmarshal([]byte(listRaw), &listOut); err != nil {
		t.Fatalf("unmarshal list output: %v", err)
	}
	if len(listOut.Patterns) == 0 || listOut.Patterns[0].Name != "point-buffer-count" {
		t.Fatalf("expected matching pattern in list output, got %+v", listOut)
	}

	renderRaw, err := tool.InvokableRun(context.Background(), `{"action":"render","pattern":"point-buffer-count","values":{"target_table":"schools","lon":"116.397","lat":"39.908","distance_m":"500"}}`)
	if err != nil {
		t.Fatalf("render InvokableRun() error = %v", err)
	}
	var renderOut GeoSQLCodebookOutput
	if err := json.Unmarshal([]byte(renderRaw), &renderOut); err != nil {
		t.Fatalf("unmarshal render output: %v", err)
	}
	if renderOut.SQL == "" || renderOut.Pattern != "point-buffer-count" {
		t.Fatalf("expected rendered SQL output, got %+v", renderOut)
	}
}
