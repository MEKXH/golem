package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileTool(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewWriteFileTool(tmpDir)
	if err != nil {
		t.Fatalf("NewWriteFileTool error: %v", err)
	}

	targetFile := filepath.Join(tmpDir, "output.txt")
	content := "hello world\nsecond line"
	argsJSON := fmt.Sprintf(`{"path": %q, "content": %q}`, targetFile, content)

	ctx := context.Background()
	result, err := tool.InvokableRun(ctx, argsJSON)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}

	if !strings.Contains(result, "success") && !strings.Contains(result, "Success") && !strings.Contains(result, "successfully") {
		t.Errorf("expected success message, got: %s", result)
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected file content %q, got %q", content, string(data))
	}
}

func TestListDirTool(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files and a subdirectory
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("b"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	tool, err := NewListDirTool(tmpDir)
	if err != nil {
		t.Fatalf("NewListDirTool error: %v", err)
	}

	ctx := context.Background()
	argsJSON := fmt.Sprintf(`{"path": %q}`, tmpDir)

	result, err := tool.InvokableRun(ctx, argsJSON)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}

	// The result should be a JSON array of strings
	if !strings.Contains(result, "file1.txt") {
		t.Errorf("expected result to contain 'file1.txt', got: %s", result)
	}
	if !strings.Contains(result, "file2.txt") {
		t.Errorf("expected result to contain 'file2.txt', got: %s", result)
	}
	if !strings.Contains(result, "subdir/") {
		t.Errorf("expected result to contain 'subdir/' (with trailing slash), got: %s", result)
	}
}

func TestReadFile_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewReadFileTool(tmpDir)
	if err != nil {
		t.Fatalf("NewReadFileTool error: %v", err)
	}

	// Attempt to read a path outside workspace using .. traversal
	outsidePath := filepath.Join(tmpDir, "..", "outside.txt")
	argsJSON := fmt.Sprintf(`{"path": %q}`, outsidePath)

	ctx := context.Background()
	_, err = tool.InvokableRun(ctx, argsJSON)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") && !strings.Contains(err.Error(), "outside workspace") {
		t.Errorf("expected access denied error, got: %v", err)
	}
}

func TestWriteFile_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewWriteFileTool(tmpDir)
	if err != nil {
		t.Fatalf("NewWriteFileTool error: %v", err)
	}

	outsidePath := filepath.Join(tmpDir, "..", "evil.txt")
	argsJSON := fmt.Sprintf(`{"path": %q, "content": "malicious"}`, outsidePath)

	ctx := context.Background()
	_, err = tool.InvokableRun(ctx, argsJSON)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") && !strings.Contains(err.Error(), "outside workspace") {
		t.Errorf("expected access denied error, got: %v", err)
	}
}

func TestListDir_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewListDirTool(tmpDir)
	if err != nil {
		t.Fatalf("NewListDirTool error: %v", err)
	}

	outsidePath := filepath.Join(tmpDir, "..")
	argsJSON := fmt.Sprintf(`{"path": %q}`, outsidePath)

	ctx := context.Background()
	_, err = tool.InvokableRun(ctx, argsJSON)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") && !strings.Contains(err.Error(), "outside workspace") {
		t.Errorf("expected access denied error, got: %v", err)
	}
}

func TestReadFile_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "readable.txt")
	os.WriteFile(testFile, []byte("content here"), 0644)

	// Empty workspace means no path restriction (backward compat)
	tool, err := NewReadFileTool("")
	if err != nil {
		t.Fatalf("NewReadFileTool error: %v", err)
	}

	ctx := context.Background()
	argsJSON := fmt.Sprintf(`{"path": %q}`, testFile)

	result, err := tool.InvokableRun(ctx, argsJSON)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}

	// Parse result to verify content
	var output ReadFileOutput
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		// If it doesn't parse as ReadFileOutput, just check raw string
		if !strings.Contains(result, "content here") {
			t.Errorf("expected result to contain 'content here', got: %s", result)
		}
		return
	}
	if !strings.Contains(output.Content, "content here") {
		t.Errorf("expected content to contain 'content here', got: %s", output.Content)
	}
}

func TestReadFile_SymlinkEscapeBlocked(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	secretFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("top-secret"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	linkPath := filepath.Join(workspace, "link-out")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink not supported on this environment: %v", err)
	}

	tool, err := NewReadFileTool(workspace)
	if err != nil {
		t.Fatalf("NewReadFileTool error: %v", err)
	}

	argsJSON := fmt.Sprintf(`{"path": %q}`, filepath.Join(linkPath, "secret.txt"))
	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Fatal("expected error for symlink escape, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("expected access denied, got: %v", err)
	}
}

func TestWriteFile_SymlinkEscapeBlocked(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()

	linkPath := filepath.Join(workspace, "link-out")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink not supported on this environment: %v", err)
	}

	tool, err := NewWriteFileTool(workspace)
	if err != nil {
		t.Fatalf("NewWriteFileTool error: %v", err)
	}

	target := filepath.Join(linkPath, "evil.txt")
	argsJSON := fmt.Sprintf(`{"path": %q, "content": "malicious"}`, target)
	_, err = tool.InvokableRun(context.Background(), argsJSON)
	if err == nil {
		t.Fatal("expected error for symlink escape, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("expected access denied, got: %v", err)
	}
}
