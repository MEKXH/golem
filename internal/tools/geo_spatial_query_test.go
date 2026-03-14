package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPrepareGeoSpatialSQL_AllowsReadonlyStatements(t *testing.T) {
	tests := []struct {
		name  string
		sql   string
		check func(t *testing.T, prepared string)
	}{
		{
			name: "select gets wrapped with limit",
			sql:  "SELECT id, name FROM parcels;",
			check: func(t *testing.T, prepared string) {
				if !strings.Contains(prepared, "LIMIT 26") {
					t.Fatalf("expected wrapped query to include LIMIT 26, got %q", prepared)
				}
				if !strings.Contains(prepared, "FROM (SELECT id, name FROM parcels)") {
					t.Fatalf("expected query wrapper, got %q", prepared)
				}
			},
		},
		{
			name: "with gets wrapped with limit",
			sql:  "WITH filtered AS (SELECT * FROM parcels) SELECT * FROM filtered",
			check: func(t *testing.T, prepared string) {
				if !strings.Contains(prepared, "LIMIT 26") {
					t.Fatalf("expected wrapped query to include LIMIT 26, got %q", prepared)
				}
			},
		},
		{
			name: "explain stays explain",
			sql:  "EXPLAIN SELECT * FROM parcels",
			check: func(t *testing.T, prepared string) {
				if strings.Contains(prepared, "LIMIT 25") {
					t.Fatalf("did not expect explain query to be rewritten with LIMIT, got %q", prepared)
				}
				if !strings.HasPrefix(prepared, "EXPLAIN") {
					t.Fatalf("expected EXPLAIN query, got %q", prepared)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prepared, err := prepareGeoSpatialSQL(tc.sql, 25)
			if err != nil {
				t.Fatalf("prepareGeoSpatialSQL() error = %v", err)
			}
			tc.check(t, prepared)
		})
	}
}

func TestPrepareGeoSpatialSQL_RejectsUnsafeStatements(t *testing.T) {
	tests := []string{
		"INSERT INTO parcels VALUES (1)",
		"UPDATE parcels SET name = 'x'",
		"DELETE FROM parcels",
		"DROP TABLE parcels",
		"COPY parcels TO STDOUT",
		"SELECT * FROM parcels; SELECT * FROM roads",
	}

	for _, sqlText := range tests {
		t.Run(sqlText, func(t *testing.T) {
			if _, err := prepareGeoSpatialSQL(sqlText, 10); err == nil {
				t.Fatalf("expected unsafe SQL to be rejected: %q", sqlText)
			}
		})
	}
}

func TestGeoSpatialQueryTool_QueryTruncatesResults(t *testing.T) {
	tool, recorder := newTestGeoSpatialQueryTool(t, GeoSpatialQueryOutput{
		Action: "query",
		SQL:    "SELECT * FROM parcels",
		Columns: []string{
			"id",
			"name",
		},
		Rows: []map[string]any{
			{"id": int64(1), "name": "a"},
			{"id": int64(2), "name": "b"},
		},
		RowCount:  1,
		Truncated: true,
	})

	out, err := tool.execute(context.Background(), &GeoSpatialQueryInput{
		Action: "query",
		SQL:    "SELECT * FROM parcels",
	})
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if recorder.lastSQL == "" || !strings.Contains(recorder.lastSQL, "LIMIT 2") {
		t.Fatalf("expected limited SQL to be executed, got %q", recorder.lastSQL)
	}
	if out.RowCount != 1 || !out.Truncated {
		t.Fatalf("expected truncated output, got %+v", out)
	}
}

func TestGeoSpatialQueryTool_SchemaUsesIntrospectionSQL(t *testing.T) {
	tool, recorder := newTestGeoSpatialQueryTool(t, GeoSpatialQueryOutput{
		Action:  "schema",
		SQL:     geoSpatialSchemaSQL,
		Columns: []string{"table_schema", "table_name"},
		Rows: []map[string]any{
			{"table_schema": "public", "table_name": "parcels"},
		},
		RowCount: 1,
	})

	out, err := tool.execute(context.Background(), &GeoSpatialQueryInput{Action: "schema"})
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if !strings.Contains(recorder.lastSQL, "geometry_columns") {
		t.Fatalf("expected schema introspection SQL, got %q", recorder.lastSQL)
	}
	if out.Action != "schema" || out.RowCount != 1 {
		t.Fatalf("unexpected schema output: %+v", out)
	}
}

func TestGeoSpatialQueryTool_RequiresQuerySQL(t *testing.T) {
	tool, _ := newTestGeoSpatialQueryTool(t, GeoSpatialQueryOutput{})

	_, err := tool.execute(context.Background(), &GeoSpatialQueryInput{Action: "query"})
	if err == nil {
		t.Fatal("expected query action without SQL to fail")
	}
}

type testGeoQueryOpener struct {
	output  GeoSpatialQueryOutput
	lastSQL string
}

func newTestGeoSpatialQueryTool(t *testing.T, output GeoSpatialQueryOutput) (*geoSpatialQueryToolImpl, *testGeoQueryOpener) {
	t.Helper()
	recorder := &testGeoQueryOpener{output: output}
	tool := &geoSpatialQueryToolImpl{
		timeoutSec: 30,
		maxRows:    1,
		readOnly:   true,
		openSession: func(ctx context.Context, readonly bool) (geoQuerySession, error) {
			return &testGeoQuerySession{recorder: recorder}, nil
		},
	}
	return tool, recorder
}

type testGeoQuerySession struct {
	recorder *testGeoQueryOpener
}

func (s *testGeoQuerySession) QueryContext(ctx context.Context, query string, args ...any) (geoQueryRows, error) {
	s.recorder.lastSQL = query
	return &testGeoQueryRows{output: s.recorder.output}, nil
}

func (s *testGeoQuerySession) Close() error {
	return nil
}

type testGeoQueryRows struct {
	output GeoSpatialQueryOutput
	index  int
}

func (r *testGeoQueryRows) Columns() ([]string, error) {
	return r.output.Columns, nil
}

func (r *testGeoQueryRows) Next() bool {
	return r.index < len(r.output.Rows)
}

func (r *testGeoQueryRows) Scan(dest ...any) error {
	if r.index >= len(r.output.Rows) {
		return errors.New("scan past end")
	}
	row := r.output.Rows[r.index]
	for i, col := range r.output.Columns {
		value, ok := row[col]
		if !ok {
			return errors.New("missing column value")
		}
		ptr, ok := dest[i].(*any)
		if !ok {
			return errors.New("unexpected scan destination")
		}
		*ptr = value
	}
	r.index++
	return nil
}

func (r *testGeoQueryRows) Err() error {
	return nil
}

func (r *testGeoQueryRows) Close() error {
	return nil
}
