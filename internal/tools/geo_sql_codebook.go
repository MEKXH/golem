package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/geocodebook"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GeoSQLCodebookInput defines the input for the geo_sql_codebook tool.
type GeoSQLCodebookInput struct {
	Action  string            `json:"action" jsonschema:"required,description=Codebook operation: list or render"`
	Intent  string            `json:"intent" jsonschema:"description=Freeform intent used to rank matching patterns"`
	Pattern string            `json:"pattern" jsonschema:"description=Named codebook pattern to render"`
	Values  map[string]string `json:"values" jsonschema:"description=Variable values for template rendering"`
	Limit   int               `json:"limit" jsonschema:"description=Maximum number of matches to return for action=list"`
}

// GeoSQLCodebookOutput defines the output for the geo_sql_codebook tool.
type GeoSQLCodebookOutput struct {
	Action   string              `json:"action"`
	Intent   string              `json:"intent,omitempty"`
	Pattern  string              `json:"pattern,omitempty"`
	Patterns []geocodebook.Match `json:"patterns,omitempty"`
	SQL      string              `json:"sql,omitempty"`
	Values   map[string]string   `json:"values,omitempty"`
	Verified bool                `json:"verified,omitempty"`
	Source   string              `json:"source,omitempty"`
}

type geoSQLCodebookToolImpl struct {
	loader *geocodebook.Loader
}

func (t *geoSQLCodebookToolImpl) execute(ctx context.Context, input *GeoSQLCodebookInput) (*GeoSQLCodebookOutput, error) {
	_ = ctx
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}

	switch action {
	case "list":
		matches, err := t.loader.ListPatterns(input.Intent, input.Limit)
		if err != nil {
			return nil, err
		}
		return &GeoSQLCodebookOutput{Action: action, Intent: input.Intent, Patterns: matches}, nil
	case "render":
		patternName := strings.TrimSpace(input.Pattern)
		if patternName == "" {
			return nil, fmt.Errorf("pattern is required when action=render")
		}
		rendered, err := t.loader.RenderPattern(patternName, input.Values)
		if err != nil {
			return nil, err
		}
		return &GeoSQLCodebookOutput{
			Action:   action,
			Pattern:  rendered.Pattern,
			SQL:      rendered.SQL,
			Values:   rendered.Variables,
			Verified: rendered.Verified,
			Source:   rendered.Source,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported action %q; expected list or render", input.Action)
	}
}

// NewGeoSQLCodebookTool creates the geo_sql_codebook tool for ranked codebook lookup and SQL rendering.
func NewGeoSQLCodebookTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &geoSQLCodebookToolImpl{loader: geocodebook.NewLoader(workspacePath)}
	return utils.InferTool(
		"geo_sql_codebook",
		"Look up verified spatial SQL patterns from the workspace geo-codebook and render them with variables before using geo_spatial_query.",
		impl.execute,
	)
}
