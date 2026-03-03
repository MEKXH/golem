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

// validatePath 验证给定路径是否在允许的工作区范围内，防止路径穿越攻击。
// 如果 workspacePath 为空，则跳过验证（仅用于向后兼容）。
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

// resolvePathBestEffort 尽力解析路径中已存在部分的符号链接。
// 对于尚不存在的写入目标，它会解析最长存在的父目录前缀。
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

// ReadFileInput 定义了 read_file 工具的输入参数。
type ReadFileInput struct {
	Path   string `json:"path" jsonschema:"required,description=Absolute path to the file"`
	Offset int    `json:"offset" jsonschema:"description=Starting line number (0-based)"`
	Limit  int    `json:"limit" jsonschema:"description=Maximum number of lines to read"`
}

// ReadFileOutput 定义了 read_file 工具的执行结果。
type ReadFileOutput struct {
	Content    string `json:"content"`     // 文件内容
	TotalLines int    `json:"total_lines"` // 文件总行数
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

// NewReadFileTool 创建 read_file 工具实例，用于读取工作区内的文件内容。
func NewReadFileTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &readFileToolImpl{workspacePath: workspacePath}
	return utils.InferTool("read_file", "Read the contents of a file", impl.execute)
}

// WriteFileInput 定义了 write_file 工具的输入参数。
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

// NewWriteFileTool 创建 write_file 工具实例，用于在工作区内写入新文件或覆盖已有文件。
func NewWriteFileTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &writeFileToolImpl{workspacePath: workspacePath}
	return utils.InferTool("write_file", "Write content to a file", impl.execute)
}

// ListDirInput 定义了 list_dir 工具的输入参数。
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

// NewListDirTool 创建 list_dir 工具实例，用于列出目录下的所有文件和文件夹。
func NewListDirTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &listDirToolImpl{workspacePath: workspacePath}
	return utils.InferTool("list_dir", "List contents of a directory", impl.execute)
}
