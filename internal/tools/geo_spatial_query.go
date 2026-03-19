package tools

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// GeoSpatialQueryInput defines the input for the geo_spatial_query tool.
type GeoSpatialQueryInput struct {
	Action string `json:"action" jsonschema:"required,description=Operation mode: schema or query"`
	SQL    string `json:"sql" jsonschema:"description=Read-only SQL to execute when action is query"`
}

// GeoSpatialQueryOutput defines the output for the geo_spatial_query tool.
type GeoSpatialQueryOutput struct {
	Action    string           `json:"action"`
	SQL       string           `json:"sql"`
	Columns   []string         `json:"columns"`
	Rows      []map[string]any `json:"rows"`
	RowCount  int              `json:"row_count"`
	Truncated bool             `json:"truncated"`
}

const geoSpatialSchemaSQL = `
SELECT
  c.table_schema,
  c.table_name,
  c.column_name,
  c.data_type,
  COALESCE(gc.f_geometry_column IS NOT NULL, false) AS is_geometry,
  COALESCE(gc.type, '') AS geometry_type,
  COALESCE(gc.srid, 0) AS srid
FROM information_schema.columns c
JOIN information_schema.tables t
  ON t.table_schema = c.table_schema
 AND t.table_name = c.table_name
LEFT JOIN public.geometry_columns gc
  ON gc.f_table_schema = c.table_schema
 AND gc.f_table_name = c.table_name
 AND gc.f_geometry_column = c.column_name
WHERE c.table_schema NOT IN ('pg_catalog', 'information_schema')
  AND t.table_type = 'BASE TABLE'
ORDER BY c.table_schema, c.table_name, c.ordinal_position
`

var (
	geoSpatialAllowedPrefixRe = regexp.MustCompile(`(?is)^(select|with|explain)\b`)
	geoSpatialForbiddenWordRe = regexp.MustCompile(`(?is)\b(insert|update|delete|drop|alter|truncate|create|copy|grant|revoke|comment|vacuum|call|refresh|merge)\b`)
	geoSpatialExplainPrefixRe = regexp.MustCompile(`(?is)^explain\b`)
)

type geoQueryRows interface {
	Columns() ([]string, error)
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

type geoQuerySession interface {
	QueryContext(ctx context.Context, query string, args ...any) (geoQueryRows, error)
	Close() error
}

type geoSpatialQueryToolImpl struct {
	timeoutSec  int
	maxRows     int
	readOnly    bool
	openSession func(ctx context.Context, readonly bool) (geoQuerySession, error)
}

func (t *geoSpatialQueryToolImpl) execute(ctx context.Context, input *GeoSpatialQueryInput) (*GeoSpatialQueryOutput, error) {
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}

	timeout := time.Duration(t.timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var sqlText string
	switch action {
	case "schema":
		sqlText = prepareGeoSpatialSchemaSQL(t.maxRows)
	case "query":
		var err error
		sqlText, err = prepareGeoSpatialSQL(input.SQL, t.maxRows)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported action %q; expected schema or query", input.Action)
	}

	session, err := t.openSession(queryCtx, t.readOnly)
	if err != nil {
		return nil, fmt.Errorf("open PostGIS session: %w", err)
	}
	defer session.Close()

	rows, err := session.QueryContext(queryCtx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("execute PostGIS query: %w", err)
	}
	defer rows.Close()

	return collectGeoSpatialQueryOutput(action, sqlText, rows, t.maxRows)
}

func prepareGeoSpatialSchemaSQL(maxRows int) string {
	limit := maxRows + 1
	if limit <= 1 {
		limit = 201
	}
	return fmt.Sprintf("%s\nLIMIT %d", strings.TrimSpace(geoSpatialSchemaSQL), limit)
}

func prepareGeoSpatialSQL(sqlText string, maxRows int) (string, error) {
	normalized, err := normalizeGeoSpatialSQL(sqlText)
	if err != nil {
		return "", err
	}
	if !geoSpatialAllowedPrefixRe.MatchString(normalized) {
		return "", fmt.Errorf("only SELECT, WITH, or EXPLAIN statements are allowed")
	}
	if geoSpatialForbiddenWordRe.MatchString(normalized) {
		return "", fmt.Errorf("query contains forbidden write or DDL keywords")
	}
	if geoSpatialExplainPrefixRe.MatchString(normalized) {
		return normalized, nil
	}

	limit := maxRows + 1
	if limit <= 1 {
		limit = 201
	}
	return fmt.Sprintf("SELECT * FROM (%s) AS golem_geo_query LIMIT %d", normalized, limit), nil
}

func normalizeGeoSpatialSQL(sqlText string) (string, error) {
	normalized := strings.TrimSpace(sqlText)
	if normalized == "" {
		return "", fmt.Errorf("sql is required when action=query")
	}
	if strings.HasSuffix(normalized, ";") {
		normalized = strings.TrimSpace(strings.TrimSuffix(normalized, ";"))
	}
	if strings.Contains(normalized, ";") {
		return "", fmt.Errorf("multiple SQL statements are not allowed")
	}
	return normalized, nil
}

func collectGeoSpatialQueryOutput(action, sqlText string, rows geoQueryRows, maxRows int) (*GeoSpatialQueryOutput, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("read result columns: %w", err)
	}

	if maxRows <= 0 {
		maxRows = 200
	}

	result := &GeoSpatialQueryOutput{
		Action:  action,
		SQL:     sqlText,
		Columns: columns,
		Rows:    make([]map[string]any, 0, maxRows),
	}

	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range dest {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan result row: %w", err)
		}
		if len(result.Rows) == maxRows {
			result.Truncated = true
			break
		}

		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = normalizeGeoSpatialValue(values[i])
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate result rows: %w", err)
	}

	result.RowCount = len(result.Rows)
	return result, nil
}

func normalizeGeoSpatialValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format(time.RFC3339Nano)
	default:
		return value
	}
}

type sqlRowsAdapter struct {
	rows *sql.Rows
}

func (r *sqlRowsAdapter) Columns() ([]string, error) {
	return r.rows.Columns()
}

func (r *sqlRowsAdapter) Next() bool {
	return r.rows.Next()
}

func (r *sqlRowsAdapter) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRowsAdapter) Err() error {
	return r.rows.Err()
}

func (r *sqlRowsAdapter) Close() error {
	return r.rows.Close()
}

type sqlTxSession struct {
	tx *sql.Tx
}

func (s *sqlTxSession) QueryContext(ctx context.Context, query string, args ...any) (geoQueryRows, error) {
	rows, err := s.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsAdapter{rows: rows}, nil
}

func (s *sqlTxSession) Close() error {
	return s.tx.Rollback()
}

type sqlDBSession struct {
	db *sql.DB
}

func (s *sqlDBSession) QueryContext(ctx context.Context, query string, args ...any) (geoQueryRows, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsAdapter{rows: rows}, nil
}

func (s *sqlDBSession) Close() error {
	return nil
}

// NewGeoSpatialQueryTool creates the geo_spatial_query tool for PostGIS schema inspection and read-only SQL execution.
func NewGeoSpatialQueryTool(postGISDSN string, timeoutSec, maxRows int, readonly bool) (tool.InvokableTool, error) {
	postGISDSN = strings.TrimSpace(postGISDSN)
	if postGISDSN == "" {
		return nil, fmt.Errorf("postgis DSN is required")
	}

	db, err := sql.Open("pgx", postGISDSN)
	if err != nil {
		return nil, fmt.Errorf("open PostGIS connection: %w", err)
	}

	impl := &geoSpatialQueryToolImpl{
		timeoutSec: timeoutSec,
		maxRows:    maxRows,
		readOnly:   readonly,
		openSession: func(ctx context.Context, readonly bool) (geoQuerySession, error) {
			if readonly {
				tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
				if err != nil {
					return nil, err
				}
				return &sqlTxSession{tx: tx}, nil
			}
			return &sqlDBSession{db: db}, nil
		},
	}

	return utils.InferTool(
		"geo_spatial_query",
		"Inspect visible PostGIS schema metadata or execute a single read-only spatial SQL query. "+
			"Use action=schema to list tables and geometry columns, or action=query to run a SELECT/WITH/EXPLAIN query.",
		impl.execute,
	)
}
