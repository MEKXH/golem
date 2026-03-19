package tools

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestGeoSpatialQueryTool_PostGISIntegration(t *testing.T) {
	dsn := os.Getenv("GOLEM_TEST_POSTGIS_DSN")
	if dsn == "" {
		t.Skip("GOLEM_TEST_POSTGIS_DSN is not set")
	}

	tool, err := NewGeoSpatialQueryTool(dsn, 30, 10, true)
	if err != nil {
		t.Fatalf("NewGeoSpatialQueryTool() error = %v", err)
	}

	queryRaw, err := tool.InvokableRun(context.Background(), `{"action":"query","sql":"SELECT PostGIS_Full_Version() AS version"}`)
	if err != nil {
		t.Fatalf("query InvokableRun() error = %v", err)
	}

	var queryOut GeoSpatialQueryOutput
	if err := json.Unmarshal([]byte(queryRaw), &queryOut); err != nil {
		t.Fatalf("unmarshal query output: %v", err)
	}
	if queryOut.RowCount != 1 {
		t.Fatalf("expected one integration row, got %+v", queryOut)
	}

	schemaRaw, err := tool.InvokableRun(context.Background(), `{"action":"schema"}`)
	if err != nil {
		t.Fatalf("schema InvokableRun() error = %v", err)
	}

	var schemaOut GeoSpatialQueryOutput
	if err := json.Unmarshal([]byte(schemaRaw), &schemaOut); err != nil {
		t.Fatalf("unmarshal schema output: %v", err)
	}
	if schemaOut.Action != "schema" {
		t.Fatalf("expected schema action output, got %+v", schemaOut)
	}
}
