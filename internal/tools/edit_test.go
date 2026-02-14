package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFileTool_ReplacesSingleMatch(t *testing.T) {
	workspace := t.TempDir()
	target := filepath.Join(workspace, "main.go")
	original := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	if err := os.WriteFile(target, []byte(original), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tool, err := NewEditFileTool(workspace)
	if err != nil {
		t.Fatalf("NewEditFileTool error: %v", err)
	}

	argsJSON := fmt.Sprintf(
		`{"path": %q, "old_text": %q, "new_text": %q}`,
		target,
		`println("hello")`,
		`println("hi")`,
	)
	if _, err := tool.InvokableRun(context.Background(), argsJSON); err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(got), `println("hi")`) {
		t.Fatalf("expected replacement applied, got content: %s", string(got))
	}
	if strings.Contains(string(got), `println("hello")`) {
		t.Fatalf("expected old text removed, got content: %s", string(got))
	}
}

func TestEditFileTool_BlocksPathTraversal(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewEditFileTool(workspace)
	if err != nil {
		t.Fatalf("NewEditFileTool error: %v", err)
	}

	outside := filepath.Join(workspace, "..", "evil.txt")
	argsJSON := fmt.Sprintf(
		`{"path": %q, "old_text": "a", "new_text": "b"}`,
		outside,
	)
	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("expected access denied, got: %v", err)
	}
}

func TestAppendFileTool_AppendsContent(t *testing.T) {
	workspace := t.TempDir()
	target := filepath.Join(workspace, "notes.txt")
	if err := os.WriteFile(target, []byte("line1\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tool, err := NewAppendFileTool(workspace)
	if err != nil {
		t.Fatalf("NewAppendFileTool error: %v", err)
	}

	argsJSON := fmt.Sprintf(`{"path": %q, "content": %q}`, target, "line2\n")
	if _, err := tool.InvokableRun(context.Background(), argsJSON); err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "line1\nline2\n" {
		t.Fatalf("expected appended content, got %q", string(got))
	}
}

func TestAppendFileTool_RejectsEmptyContent(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewAppendFileTool(workspace)
	if err != nil {
		t.Fatalf("NewAppendFileTool error: %v", err)
	}

	target := filepath.Join(workspace, "notes.txt")
	argsJSON := fmt.Sprintf(`{"path": %q, "content": "   "}`, target)
	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Fatal("expected empty content error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "empty") {
		t.Fatalf("expected empty-content error, got: %v", err)
	}
}

func TestAppendFileTool_BlocksPathTraversal(t *testing.T) {
	workspace := t.TempDir()
	tool, err := NewAppendFileTool(workspace)
	if err != nil {
		t.Fatalf("NewAppendFileTool error: %v", err)
	}

	outside := filepath.Join(workspace, "..", "evil.txt")
	argsJSON := fmt.Sprintf(`{"path": %q, "content": "malicious"}`, outside)
	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("expected access denied, got: %v", err)
	}
}
