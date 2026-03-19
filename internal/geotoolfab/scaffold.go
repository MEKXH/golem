package geotoolfab

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScaffoldSpec describes the desired fabricated geo tool skeleton.
type ScaffoldSpec struct {
	Name        string
	Description string
	Runner      string
	Parameters  map[string]Parameter
}

// Scaffold contains a dry-run fabricated geo tool skeleton.
type Scaffold struct {
	ToolName         string
	ManifestPath     string
	ScriptPath       string
	ManifestBody     string
	ScriptBody       string
	ValidationPassed bool
}

// BuildScaffold creates a validator-compliant fabricated geo tool skeleton without writing files.
func BuildScaffold(workspacePath string, spec ScaffoldSpec) (*Scaffold, error) {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return nil, fmt.Errorf("workspace path is required")
	}
	workspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace path: %w", err)
	}

	toolName := normalizeScaffoldToolName(spec.Name)
	if !validToolNamePattern.MatchString(toolName) {
		return nil, fmt.Errorf("fabricated geo tool must use a geo_* tool name, got %q", toolName)
	}

	runner := strings.TrimSpace(spec.Runner)
	if runner == "" {
		runner = "python"
	}
	description := strings.TrimSpace(spec.Description)
	if description == "" {
		description = "Workspace-fabricated geo tool."
	}

	normalizedParams := make(map[string]Parameter, len(spec.Parameters))
	for name, param := range spec.Parameters {
		paramName := strings.TrimSpace(name)
		if paramName == "" {
			return nil, fmt.Errorf("fabricated geo tool %q contains an empty parameter name", toolName)
		}
		paramType := normalizeParameterType(param.Type)
		if !isSupportedParameterType(paramType) {
			return nil, fmt.Errorf("fabricated geo tool %q parameter %q has unsupported type %q", toolName, paramName, param.Type)
		}
		normalizedParams[paramName] = Parameter{
			Type:        paramType,
			Description: strings.TrimSpace(param.Description),
			Required:    param.Required,
		}
	}

	manifestPath := filepath.Join(workspacePath, "tools", "geo", toolName+".yaml")
	scriptPath := filepath.Join(workspacePath, "tools", "geo", "scripts", toolName+".py")
	scriptBody := buildPythonScriptBody(toolName)

	if err := validateScaffoldDefinition(toolName, description, runner, normalizedParams, scriptBody); err != nil {
		return nil, err
	}

	return &Scaffold{
		ToolName:         toolName,
		ManifestPath:     manifestPath,
		ScriptPath:       scriptPath,
		ManifestBody:     buildManifestBody(toolName, description, runner, normalizedParams),
		ScriptBody:       scriptBody,
		ValidationPassed: true,
	}, nil
}

func validateScaffoldDefinition(name, description, runner string, parameters map[string]Parameter, scriptBody string) error {
	tempDir, err := os.MkdirTemp("", "golem-geo-scaffold-")
	if err != nil {
		return fmt.Errorf("create scaffold temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempScriptPath := filepath.Join(tempDir, name+".py")
	if err := os.WriteFile(tempScriptPath, []byte(scriptBody), 0o644); err != nil {
		return fmt.Errorf("write scaffold temp script: %w", err)
	}

	_, err = ValidateDefinition(Definition{
		Name:           name,
		Description:    description,
		Runner:         runner,
		ScriptPath:     tempScriptPath,
		WorkingDir:     tempDir,
		Parameters:     parameters,
		TimeoutSeconds: defaultTimeoutSeconds,
	})
	return err
}

func normalizeScaffoldToolName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.TrimPrefix(name, "geo_")
	if name == "" {
		name = "tool"
	}
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	clean := strings.Trim(b.String(), "_")
	if clean == "" {
		clean = "tool"
	}
	return "geo_" + clean
}

func buildManifestBody(name, description, runner string, parameters map[string]Parameter) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("name: %s\n", name))
	sb.WriteString(fmt.Sprintf("description: %s\n", description))
	sb.WriteString(fmt.Sprintf("runner: %s\n", runner))
	sb.WriteString(fmt.Sprintf("script: tools/geo/scripts/%s.py\n", name))
	sb.WriteString("parameters:\n")

	keys := make([]string, 0, len(parameters))
	for key := range parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		param := parameters[key]
		sb.WriteString(fmt.Sprintf("  %s:\n", key))
		sb.WriteString(fmt.Sprintf("    type: %s\n", param.Type))
		if param.Description != "" {
			sb.WriteString(fmt.Sprintf("    description: %s\n", param.Description))
		}
		if param.Required {
			sb.WriteString("    required: true\n")
		}
	}
	return sb.String()
}

func buildPythonScriptBody(name string) string {
	return fmt.Sprintf("import json\nimport sys\n\n\ndef main():\n    payload = json.load(sys.stdin)\n    result = {\n        \"tool\": \"%s\",\n        \"input\": payload,\n        \"status\": \"todo\"\n    }\n    json.dump(result, sys.stdout)\n\n\nif __name__ == \"__main__\":\n    main()\n", name)
}
