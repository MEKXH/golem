package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/geotoolfab"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type geoFabricatedTool struct {
	def geotoolfab.Definition
}

type geoFabricatedInvocation struct {
	payload string
	args    []string
	timeout time.Duration
}

// LoadGeoFabricatedTools loads workspace-defined geo tools and returns them as invokable tools.
func LoadGeoFabricatedTools(workspacePath string) ([]tool.InvokableTool, error) {
	defs, err := geotoolfab.NewLoader(workspacePath).Load()
	if err != nil {
		return nil, err
	}

	result := make([]tool.InvokableTool, 0, len(defs))
	for _, def := range defs {
		toolImpl, err := NewGeoFabricatedTool(def)
		if err != nil {
			return nil, err
		}
		result = append(result, toolImpl)
	}
	return result, nil
}

// NewGeoFabricatedTool wraps a fabricated geo tool definition as an invokable tool.
func NewGeoFabricatedTool(def geotoolfab.Definition) (tool.InvokableTool, error) {
	validated, err := geotoolfab.ValidateDefinition(def)
	if err != nil {
		return nil, err
	}
	return &geoFabricatedTool{def: validated}, nil
}

func (t *geoFabricatedTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	_ = ctx
	params := make(map[string]*schema.ParameterInfo, len(t.def.Parameters))
	for name, param := range t.def.Parameters {
		params[name] = &schema.ParameterInfo{
			Type:     schema.DataType(param.Type),
			Desc:     param.Description,
			Required: param.Required,
		}
	}

	return &schema.ToolInfo{
		Name:        t.def.Name,
		Desc:        t.def.Description,
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

func (t *geoFabricatedTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	_ = opts

	invocation, err := prepareGeoFabricatedInvocation(t.def, argumentsInJSON)
	if err != nil {
		return "", err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, invocation.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, t.def.Runner, invocation.args...)
	cmd.Dir = t.def.WorkingDir
	cmd.Stdin = strings.NewReader(invocation.payload)

	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("fabricated geo tool %q timed out after %ds", t.def.Name, t.def.TimeoutSeconds)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("fabricated geo tool %q failed: %s", t.def.Name, msg)
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		if text := strings.TrimSpace(stderr.String()); text != "" {
			return text, nil
		}
		return "{}", nil
	}
	return result, nil
}

func parseGeoFabricatedArguments(argumentsInJSON string) (map[string]any, error) {
	trimmed := strings.TrimSpace(argumentsInJSON)
	if trimmed == "" {
		trimmed = "{}"
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, fmt.Errorf("parse fabricated geo tool arguments: %w", err)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func validateGeoFabricatedArguments(input map[string]any, parameters map[string]geotoolfab.Parameter) error {
	for name, param := range parameters {
		value, exists := input[name]
		if !exists || value == nil {
			if param.Required {
				return fmt.Errorf("fabricated geo tool argument %q is required", name)
			}
			continue
		}
		if !matchesGeoFabricatedType(value, param.Type) {
			return fmt.Errorf("fabricated geo tool argument %q must be %s", name, param.Type)
		}
	}
	return nil
}

func matchesGeoFabricatedType(value any, expected string) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && math.Trunc(number) == number
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	default:
		return false
	}
}

func prepareGeoFabricatedInvocation(def geotoolfab.Definition, argumentsInJSON string) (geoFabricatedInvocation, error) {
	input, err := parseGeoFabricatedArguments(argumentsInJSON)
	if err != nil {
		return geoFabricatedInvocation{}, err
	}
	if err := validateGeoFabricatedArguments(input, def.Parameters); err != nil {
		return geoFabricatedInvocation{}, err
	}

	payload := strings.TrimSpace(argumentsInJSON)
	if payload == "" {
		payload = "{}"
	}

	timeout := time.Duration(def.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	args := append(append([]string(nil), def.Args...), def.ScriptPath)
	return geoFabricatedInvocation{
		payload: payload,
		args:    args,
		timeout: timeout,
	}, nil
}
