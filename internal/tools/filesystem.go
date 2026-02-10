package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// validatePath checks that the given path is within the workspace boundary.
// If workspacePath is empty, validation is skipped (backward compatibility).
func validatePath(path, workspacePath string) error {
	if workspacePath == "" {
		return nil
	}

	workspaceAbs, err := filepath.Abs(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace path: %w", err)
	}
	workspaceAbs = filepath.Clean(workspaceAbs)

	workspaceResolved, err := resolvePathBestEffort(workspaceAbs)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	targetAbs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	targetAbs = filepath.Clean(targetAbs)

	targetResolved, err := resolvePathBestEffort(targetAbs)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	if !isWithinWorkspace(targetResolved, workspaceResolved) {
		return fmt.Errorf("access denied: path %q is outside workspace %q", targetResolved, workspaceResolved)
	}
	return nil
}

func isWithinWorkspace(target, workspace string) bool {
	rel, err := filepath.Rel(workspace, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// resolvePathBestEffort resolves symlinks for the longest existing prefix of the path.
// This protects write targets where the final path may not exist yet.
func resolvePathBestEffort(path string) (string, error) {
	path = filepath.Clean(path)

	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	current := path
	var missing []string
	for {
		if _, err := os.Lstat(current); err == nil {
			resolvedPrefix, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			resolved := filepath.Clean(resolvedPrefix)
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("failed to find existing parent for path: %s", path)
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

// ReadFileInput parameters for read_file tool
type ReadFileInput struct {
	Path   string `json:"path" jsonschema:"required,description=Absolute path to the file"`
	Offset int    `json:"offset" jsonschema:"description=Starting line number (0-based)"`
	Limit  int    `json:"limit" jsonschema:"description=Maximum number of lines to read"`
}

// ReadFileOutput result of read_file tool
type ReadFileOutput struct {
	Content    string `json:"content"`
	TotalLines int    `json:"total_lines"`
}

type readFileToolImpl struct {
	workspacePath string
}

func (t *readFileToolImpl) execute(ctx context.Context, input *ReadFileInput) (*ReadFileOutput, error) {
	if err := validatePath(input.Path, t.workspacePath); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(input.Path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	if input.Offset > 0 {
		if input.Offset >= len(lines) {
			lines = []string{}
		} else {
			lines = lines[input.Offset:]
		}
	}

	if input.Limit > 0 && input.Limit < len(lines) {
		lines = lines[:input.Limit]
	}

	return &ReadFileOutput{
		Content:    strings.Join(lines, "\n"),
		TotalLines: totalLines,
	}, nil
}

// NewReadFileTool creates the read_file tool
func NewReadFileTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &readFileToolImpl{workspacePath: workspacePath}
	return utils.InferTool("read_file", "Read the contents of a file", impl.execute)
}

// WriteFileInput parameters for write_file tool
type WriteFileInput struct {
	Path    string `json:"path" jsonschema:"required,description=Absolute path to the file"`
	Content string `json:"content" jsonschema:"required,description=Content to write"`
}

type writeFileToolImpl struct {
	workspacePath string
}

func (t *writeFileToolImpl) execute(ctx context.Context, input *WriteFileInput) (string, error) {
	if err := validatePath(input.Path, t.workspacePath); err != nil {
		return "", err
	}

	err := os.WriteFile(input.Path, []byte(input.Content), 0644)
	if err != nil {
		return "", err
	}
	return "File written successfully", nil
}

// NewWriteFileTool creates the write_file tool
func NewWriteFileTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &writeFileToolImpl{workspacePath: workspacePath}
	return utils.InferTool("write_file", "Write content to a file", impl.execute)
}

// ListDirInput parameters for list_dir tool
type ListDirInput struct {
	Path string `json:"path" jsonschema:"required,description=Directory path to list"`
}

type listDirToolImpl struct {
	workspacePath string
}

func (t *listDirToolImpl) execute(ctx context.Context, input *ListDirInput) ([]string, error) {
	if err := validatePath(input.Path, t.workspacePath); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(input.Path)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		result = append(result, name)
	}
	return result, nil
}

// NewListDirTool creates the list_dir tool
func NewListDirTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &listDirToolImpl{workspacePath: workspacePath}
	return utils.InferTool("list_dir", "List contents of a directory", impl.execute)
}
