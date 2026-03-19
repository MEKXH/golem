package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GeoProcessInput defines the input for the geo_process tool.
type GeoProcessInput struct {
	Command string   `json:"command" jsonschema:"required,description=GDAL command name (e.g. gdal_translate, gdalwarp, ogr2ogr)"`
	Args    []string `json:"args" jsonschema:"required,description=Command arguments as a list of strings"`
}

// GeoProcessOutput defines the output of the geo_process tool.
type GeoProcessOutput struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

type geoProcessToolImpl struct {
	gdalBinDir          string
	workspaceDir        string
	timeoutSec          int
	restrictToWorkspace bool
}

func (t *geoProcessToolImpl) execute(ctx context.Context, input *GeoProcessInput) (*GeoProcessOutput, error) {
	command := strings.TrimSpace(input.Command)
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Enforce whitelist
	if !gdalAllowedCommands[command] {
		return nil, fmt.Errorf("command %q is not allowed; permitted commands: %s", command, gdalAllowedCommandList())
	}

	if len(input.Args) == 0 {
		return nil, fmt.Errorf("args is required")
	}

	// Check for shell injection
	if hasMeta, msg := containsShellMeta(input.Args); hasMeta {
		return &GeoProcessOutput{
			Command:  command,
			ExitCode: 1,
			Stderr:   fmt.Sprintf("Blocked: %s", msg),
		}, nil
	}

	// Validate output file paths are within workspace
	if t.restrictToWorkspace && t.workspaceDir != "" {
		if err := t.validateOutputPaths(input.Args); err != nil {
			return &GeoProcessOutput{
				Command:  command,
				ExitCode: 1,
				Stderr:   fmt.Sprintf("Path restriction: %s", err.Error()),
			}, nil
		}
	}

	if !isGdalAvailable(t.gdalBinDir, command) {
		return nil, fmt.Errorf("%s", gdalNotFoundError(command))
	}

	stdout, stderr, exitCode, err := runGdalCommand(ctx, t.gdalBinDir, command, input.Args, t.workspaceDir, t.timeoutSec)
	if err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", command, err)
	}

	return &GeoProcessOutput{
		Command:  command,
		ExitCode: exitCode,
		Stdout:   strings.TrimSpace(stdout),
		Stderr:   strings.TrimSpace(stderr),
	}, nil
}

// validateOutputPaths checks that any argument that looks like an output file path
// is within the workspace directory. Heuristic: the last non-flag argument is
// typically the output path for most GDAL commands.
func (t *geoProcessToolImpl) validateOutputPaths(args []string) error {
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		// For ogr2ogr, the output is typically the first non-flag positional arg.
		// For gdal_translate/gdalwarp, the output is the last positional arg.
		// We validate all positional arguments that look like file paths with geo extensions.
		if _, isGeo := detectGeoFormat(arg); isGeo {
			if err := validateGeoFilePath(arg, t.workspaceDir, true); err != nil {
				return fmt.Errorf("argument[%d] %q: %w", i, arg, err)
			}
		}
	}
	return nil
}

func gdalAllowedCommandList() string {
	cmds := make([]string, 0, len(gdalAllowedCommands))
	for cmd := range gdalAllowedCommands {
		cmds = append(cmds, cmd)
	}
	return strings.Join(cmds, ", ")
}

// NewGeoProcessTool creates the geo_process tool for executing whitelisted GDAL commands.
func NewGeoProcessTool(gdalBinDir, workspaceDir string, timeoutSec int, restrictToWorkspace bool) (tool.InvokableTool, error) {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	impl := &geoProcessToolImpl{
		gdalBinDir:          gdalBinDir,
		workspaceDir:        workspaceDir,
		timeoutSec:          timeoutSec,
		restrictToWorkspace: restrictToWorkspace,
	}
	return utils.InferTool(
		"geo_process",
		"Execute a whitelisted GDAL command (e.g. gdal_translate, gdalwarp, ogr2ogr) with given arguments. "+
			"Use for format conversion, reprojection, clipping, and other raster/vector processing.",
		impl.execute,
	)
}
