package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryTools_ReadWrite(t *testing.T) {
	workspace := t.TempDir()

	writeTool, err := NewWriteMemoryTool(workspace)
	if err != nil {
		t.Fatalf("NewWriteMemoryTool error: %v", err)
	}
	readTool, err := NewReadMemoryTool(workspace)
	if err != nil {
		t.Fatalf("NewReadMemoryTool error: %v", err)
	}

	ctx := context.Background()
	writeArgs := `{"content":"important memory"}`
	if _, err := writeTool.InvokableRun(ctx, writeArgs); err != nil {
		t.Fatalf("write memory error: %v", err)
	}

	result, err := readTool.InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("read memory error: %v", err)
	}
	if !strings.Contains(result, "important memory") {
		t.Fatalf("expected written memory in output, got: %s", result)
	}
}

func TestAppendDiaryTool(t *testing.T) {
	workspace := t.TempDir()
	appendTool, err := NewAppendDiaryTool(workspace)
	if err != nil {
		t.Fatalf("NewAppendDiaryTool error: %v", err)
	}

	ctx := context.Background()
	if _, err := appendTool.InvokableRun(ctx, `{"entry":"today summary"}`); err != nil {
		t.Fatalf("append diary error: %v", err)
	}

	date := strings.Split(strings.Split(fmt.Sprintf("%s", filepath.Base(filepath.Dir(filepath.Join(workspace, "memory", "x")))), "/")[0], "/")[0]
	_ = date // keep deterministic path composition out of the assertion

	files, err := os.ReadDir(filepath.Join(workspace, "memory"))
	if err != nil {
		t.Fatalf("ReadDir memory: %v", err)
	}
	foundDiary := false
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".md") && f.Name() != "MEMORY.md" {
			foundDiary = true
		}
	}
	if !foundDiary {
		t.Fatal("expected a diary markdown file to be created")
	}
}
