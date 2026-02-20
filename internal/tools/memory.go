package tools

import (
	"context"

	"github.com/MEKXH/golem/internal/memory"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type ReadMemoryInput struct{}

type ReadMemoryOutput struct {
	Content string `json:"content"`
}

type readMemoryToolImpl struct {
	manager *memory.Manager
}

func (t *readMemoryToolImpl) execute(ctx context.Context, input *ReadMemoryInput) (*ReadMemoryOutput, error) {
	content, err := t.manager.ReadLongTerm()
	if err != nil {
		return nil, err
	}
	return &ReadMemoryOutput{Content: content}, nil
}

func NewReadMemoryTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &readMemoryToolImpl{manager: memory.NewManager(workspacePath)}
	return utils.InferTool("read_memory", "Read long-term memory from memory/MEMORY.md", impl.execute)
}

type WriteMemoryInput struct {
	Content string `json:"content" jsonschema:"required,description=Content to store as long-term memory"`
}

type writeMemoryToolImpl struct {
	manager *memory.Manager
}

func (t *writeMemoryToolImpl) execute(ctx context.Context, input *WriteMemoryInput) (string, error) {
	if err := t.manager.WriteLongTerm(input.Content); err != nil {
		return "", err
	}
	return "Memory updated successfully", nil
}

func NewWriteMemoryTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &writeMemoryToolImpl{manager: memory.NewManager(workspacePath)}
	return utils.InferTool("write_memory", "Write long-term memory to memory/MEMORY.md", impl.execute)
}

type AppendDiaryInput struct {
	Entry string `json:"entry" jsonschema:"required,description=Diary entry content to append"`
}

type AppendDiaryOutput struct {
	DiaryPath string `json:"diary_path"`
}

type appendDiaryToolImpl struct {
	manager *memory.Manager
}

func (t *appendDiaryToolImpl) execute(ctx context.Context, input *AppendDiaryInput) (*AppendDiaryOutput, error) {
	path, err := t.manager.AppendDiary(input.Entry)
	if err != nil {
		return nil, err
	}
	return &AppendDiaryOutput{DiaryPath: path}, nil
}

func NewAppendDiaryTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &appendDiaryToolImpl{manager: memory.NewManager(workspacePath)}
	return utils.InferTool("append_diary", "Append a diary entry under memory/YYYY-MM-DD.md", impl.execute)
}
