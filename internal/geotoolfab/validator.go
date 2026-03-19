package geotoolfab

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateDefinition normalizes and validates one fabricated geo tool definition.
func ValidateDefinition(def Definition) (Definition, error) {
	name := strings.TrimSpace(def.Name)
	if !validToolNamePattern.MatchString(name) {
		return Definition{}, fmt.Errorf("fabricated geo tool must use a geo_* tool name, got %q", def.Name)
	}

	description := strings.TrimSpace(def.Description)
	if description == "" {
		return Definition{}, fmt.Errorf("fabricated geo tool %q description is required", name)
	}

	runner := strings.TrimSpace(def.Runner)
	if runner == "" {
		runner = "python"
	}

	scriptPath := strings.TrimSpace(def.ScriptPath)
	if scriptPath == "" {
		return Definition{}, fmt.Errorf("fabricated geo tool %q script path is required", name)
	}
	info, err := os.Stat(scriptPath)
	if err != nil {
		return Definition{}, fmt.Errorf("fabricated geo tool %q script path invalid: %w", name, err)
	}
	if info.IsDir() {
		return Definition{}, fmt.Errorf("fabricated geo tool %q script path must be a file: %s", name, scriptPath)
	}

	workingDir := strings.TrimSpace(def.WorkingDir)
	if workingDir != "" {
		workingDirInfo, err := os.Stat(workingDir)
		if err != nil {
			return Definition{}, fmt.Errorf("fabricated geo tool %q working_dir invalid: %w", name, err)
		}
		if !workingDirInfo.IsDir() {
			return Definition{}, fmt.Errorf("fabricated geo tool %q working_dir must be a directory: %s", name, workingDir)
		}
	}

	parameters := make(map[string]Parameter, len(def.Parameters))
	for key, param := range def.Parameters {
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

	timeoutSeconds := def.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeoutSeconds
	}

	return Definition{
		Name:           name,
		Description:    description,
		Runner:         runner,
		ScriptPath:     scriptPath,
		Args:           append([]string(nil), def.Args...),
		WorkingDir:     workingDir,
		Parameters:     parameters,
		TimeoutSeconds: timeoutSeconds,
		SourcePath:     strings.TrimSpace(def.SourcePath),
	}, nil
}

func defaultWorkingDirForScript(scriptPath string) string {
	if strings.TrimSpace(scriptPath) == "" {
		return ""
	}
	return filepath.Dir(scriptPath)
}
