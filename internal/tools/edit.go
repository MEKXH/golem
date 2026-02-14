package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// EditFileInput parameters for edit_file tool.
type EditFileInput struct {
	Path    string `json:"path" jsonschema:"required,description=Absolute path to the file"`
	OldText string `json:"old_text" jsonschema:"required,description=Exact existing text to replace"`
	NewText string `json:"new_text" jsonschema:"required,description=Replacement text"`
}

type editFileToolImpl struct {
	workspacePath string
}

func (t *editFileToolImpl) execute(ctx context.Context, input *EditFileInput) (string, error) {
	if err := validatePath(input.Path, t.workspacePath); err != nil {
		return "", err
	}
	if input.OldText == "" {
		return "", fmt.Errorf("old_text must not be empty")
	}

	data, err := os.ReadFile(input.Path)
	if err != nil {
		return "", err
	}
	content := string(data)
	occurrences := strings.Count(content, input.OldText)
	if occurrences == 0 {
		return "", fmt.Errorf("old_text not found in file")
	}
	if occurrences > 1 {
		return "", fmt.Errorf("old_text matches multiple locations (%d); provide a unique snippet", occurrences)
	}

	updated := strings.Replace(content, input.OldText, input.NewText, 1)
	if err := os.WriteFile(input.Path, []byte(updated), 0644); err != nil {
		return "", err
	}
	return "File edited successfully", nil
}

// NewEditFileTool creates the edit_file tool.
func NewEditFileTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &editFileToolImpl{workspacePath: workspacePath}
	return utils.InferTool("edit_file", "Edit one exact snippet in a file via old_text -> new_text replacement", impl.execute)
}

// AppendFileInput parameters for append_file tool.
type AppendFileInput struct {
	Path    string `json:"path" jsonschema:"required,description=Absolute path to the file"`
	Content string `json:"content" jsonschema:"required,description=Content to append to file end"`
}

type appendFileToolImpl struct {
	workspacePath string
}

func (t *appendFileToolImpl) execute(ctx context.Context, input *AppendFileInput) (string, error) {
	if err := validatePath(input.Path, t.workspacePath); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.Content) == "" {
		return "", fmt.Errorf("content must not be empty")
	}

	f, err := os.OpenFile(input.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.WriteString(input.Content); err != nil {
		return "", err
	}
	return "File appended successfully", nil
}

// NewAppendFileTool creates the append_file tool.
func NewAppendFileTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &appendFileToolImpl{workspacePath: workspacePath}
	return utils.InferTool("append_file", "Append content to a file", impl.execute)
}
