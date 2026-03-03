package tools

import (
	"context"

	"github.com/MEKXH/golem/internal/memory"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// ReadMemoryInput 定义了 read_memory 工具的输入参数（无参数）。
type ReadMemoryInput struct{}

// ReadMemoryOutput 定义了 read_memory 工具的执行结果。
type ReadMemoryOutput struct {
	Content string `json:"content"` // 长期记忆文件的完整内容
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

// NewReadMemoryTool 创建 read_memory 工具实例，用于从 memory/MEMORY.md 读取长期记忆。
func NewReadMemoryTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &readMemoryToolImpl{manager: memory.NewManager(workspacePath)}
	return utils.InferTool("read_memory", "Read long-term memory from memory/MEMORY.md", impl.execute)
}

// WriteMemoryInput 定义了 write_memory 工具的输入参数。
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

// NewWriteMemoryTool 创建 write_memory 工具实例，用于将内容写入 memory/MEMORY.md 作为长期记忆。
func NewWriteMemoryTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &writeMemoryToolImpl{manager: memory.NewManager(workspacePath)}
	return utils.InferTool("write_memory", "Write long-term memory to memory/MEMORY.md", impl.execute)
}

// AppendDiaryInput 定义了 append_diary 工具的输入参数。
type AppendDiaryInput struct {
	Entry string `json:"entry" jsonschema:"required,description=Diary entry content to append"`
}

// AppendDiaryOutput 定义了 append_diary 工具的执行结果。
type AppendDiaryOutput struct {
	DiaryPath string `json:"diary_path"` // 写入的日记文件路径
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

// NewAppendDiaryTool 创建 append_diary 工具实例，用于在 memory/YYYY-MM-DD.md 中追加日记分录。
func NewAppendDiaryTool(workspacePath string) (tool.InvokableTool, error) {
	impl := &appendDiaryToolImpl{manager: memory.NewManager(workspacePath)}
	return utils.InferTool("append_diary", "Append a diary entry under memory/YYYY-MM-DD.md", impl.execute)
}
