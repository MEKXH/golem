package geotoolfab

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultTimeoutSeconds = 120
	manifestDirName       = "tools/geo"
)

var validToolNamePattern = regexp.MustCompile(`^geo_[a-z0-9_]+$`)

// Parameter defines one tool input parameter declared in a workspace manifest.
type Parameter struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

// Definition is the validated, resolved form of one fabricated geo tool manifest.
type Definition struct {
	Name           string
	Description    string
	Runner         string
	ScriptPath     string
	Args           []string
	WorkingDir     string
	Parameters     map[string]Parameter
	TimeoutSeconds int
	SourcePath     string
}

type manifest struct {
	Name           string               `yaml:"name"`
	Description    string               `yaml:"description"`
	Runner         string               `yaml:"runner"`
	Script         string               `yaml:"script"`
	Args           []string             `yaml:"args"`
	WorkingDir     string               `yaml:"working_dir"`
	Parameters     map[string]Parameter `yaml:"parameters"`
	TimeoutSeconds int                  `yaml:"timeout_seconds"`
}

// Loader loads fabricated geo tool manifests from the workspace.
type Loader struct {
	workspacePath string
}

// NewLoader returns a workspace loader for fabricated geo tools.
func NewLoader(workspacePath string) *Loader {
	return &Loader{workspacePath: strings.TrimSpace(workspacePath)}
}

// Load discovers and validates all fabricated geo tool manifests.
func (l *Loader) Load() ([]Definition, error) {
	if strings.TrimSpace(l.workspacePath) == "" {
		return nil, nil
	}

	workspaceAbs, err := filepath.Abs(l.workspacePath)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace path: %w", err)
	}

	manifestDir := filepath.Join(workspaceAbs, filepath.FromSlash(manifestDirName))
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read fabricated geo tool directory: %w", err)
	}

	defs := make([]Definition, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		sourcePath := filepath.Join(manifestDir, entry.Name())
		def, err := l.loadDefinition(workspaceAbs, sourcePath)
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}

	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs, nil
}

// BuildSummary returns a system-prompt friendly summary of fabricated geo tool conventions
// and any installed workspace geo tools discovered from manifests.
func (l *Loader) BuildSummary() string {
	var sb strings.Builder
	sb.WriteString("## Geo Tool Fabrication\n\n")
	sb.WriteString("- When a reusable spatial capability is missing, create a script under `tools/geo/scripts/` and a manifest under `tools/geo/<tool_name>.yaml`.\n")
	sb.WriteString("- Fabricated geo tools must use `geo_` names and declare `name`, `description`, `runner`, `script`, and `parameters`.\n")
	sb.WriteString("- Fabricated geo tools receive their arguments as JSON on stdin and are auto-registered on the next agent startup.\n")

	defs, err := l.Load()
	if err != nil || len(defs) == 0 {
		return sb.String()
	}

	sb.WriteString("\nInstalled fabricated geo tools:\n")
	for _, def := range defs {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", def.Name, def.Description))
	}
	return sb.String()
}

func (l *Loader) loadDefinition(workspacePath, sourcePath string) (Definition, error) {
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return Definition{}, fmt.Errorf("read fabricated geo tool manifest %q: %w", sourcePath, err)
	}

	var raw manifest
	if err := yaml.Unmarshal(body, &raw); err != nil {
		return Definition{}, fmt.Errorf("parse fabricated geo tool manifest %q: %w", sourcePath, err)
	}

	name := strings.TrimSpace(raw.Name)
	if !validToolNamePattern.MatchString(name) {
		return Definition{}, fmt.Errorf("fabricated geo tool manifest %q must use a geo_* tool name, got %q", sourcePath, raw.Name)
	}

	description := strings.TrimSpace(raw.Description)
	if description == "" {
		return Definition{}, fmt.Errorf("fabricated geo tool manifest %q is missing description", sourcePath)
	}

	runner := strings.TrimSpace(raw.Runner)
	if runner == "" {
		runner = "python"
	}

	scriptPath, err := resolveWithinWorkspace(workspacePath, raw.Script)
	if err != nil {
		return Definition{}, fmt.Errorf("fabricated geo tool %q script path invalid: %w", name, err)
	}
	info, err := os.Stat(scriptPath)
	if err != nil {
		return Definition{}, fmt.Errorf("fabricated geo tool %q script path invalid: %w", name, err)
	}
	if info.IsDir() {
		return Definition{}, fmt.Errorf("fabricated geo tool %q script path must be a file: %s", name, scriptPath)
	}

	workingDir := workspacePath
	if strings.TrimSpace(raw.WorkingDir) != "" {
		workingDir, err = resolveWithinWorkspace(workspacePath, raw.WorkingDir)
		if err != nil {
			return Definition{}, fmt.Errorf("fabricated geo tool %q working_dir invalid: %w", name, err)
		}
	}

	parameters := make(map[string]Parameter, len(raw.Parameters))
	for key, param := range raw.Parameters {
		paramName := strings.TrimSpace(key)
		if paramName == "" {
			return Definition{}, fmt.Errorf("fabricated geo tool %q contains an empty parameter name", name)
		}
		paramType := normalizeParameterType(param.Type)
		if !isSupportedParameterType(paramType) {
			return Definition{}, fmt.Errorf("fabricated geo tool %q parameter %q has unsupported type %q", name, paramName, param.Type)
		}
		parameters[paramName] = Parameter{
			Type:        paramType,
			Description: strings.TrimSpace(param.Description),
			Required:    param.Required,
		}
	}

	timeoutSeconds := raw.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeoutSeconds
	}

	return Definition{
		Name:           name,
		Description:    description,
		Runner:         runner,
		ScriptPath:     scriptPath,
		Args:           append([]string(nil), raw.Args...),
		WorkingDir:     workingDir,
		Parameters:     parameters,
		TimeoutSeconds: timeoutSeconds,
		SourcePath:     sourcePath,
	}, nil
}

func resolveWithinWorkspace(workspacePath, rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", fmt.Errorf("path is required")
	}

	resolved := rawPath
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(workspacePath, filepath.FromSlash(rawPath))
	}
	resolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	workspaceAbs, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	if rel, err := filepath.Rel(workspaceAbs, resolved); err != nil {
		return "", fmt.Errorf("check workspace containment: %w", err)
	} else if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace %q", rawPath, workspaceAbs)
	}
	return resolved, nil
}

func normalizeParameterType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "string"
	}
	return value
}

func isSupportedParameterType(value string) bool {
	switch value {
	case "string", "number", "integer", "boolean", "array", "object":
		return true
	default:
		return false
	}
}
