package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSystemPrompt_IncludesRecentDiaries(t *testing.T) {
	workspace := t.TempDir()
	memDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("long-term notes"), 0644); err != nil {
		t.Fatalf("WriteFile MEMORY: %v", err)
	}
	diaries := map[string]string{
		"2026-02-08.md": "oldest",
		"2026-02-09.md": "d2",
		"2026-02-10.md": "d3",
		"2026-02-11.md": "latest",
	}
	for name, content := range diaries {
		if err := os.WriteFile(filepath.Join(memDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	cb := NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()

	if !strings.Contains(prompt, "Long-term Memory") || !strings.Contains(prompt, "long-term notes") {
		t.Fatalf("expected long-term memory in prompt, got: %s", prompt)
	}
	if strings.Contains(prompt, "oldest") {
		t.Fatalf("did not expect oldest diary in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "d2") || !strings.Contains(prompt, "d3") || !strings.Contains(prompt, "latest") {
		t.Fatalf("expected three most recent diaries in prompt, got: %s", prompt)
	}
}

func TestBuildMessages_IncludesMediaList(t *testing.T) {
	cb := NewContextBuilder(t.TempDir())
	msgs := cb.BuildMessages(nil, "analyze this", []string{"a.png", "b.txt"})
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	last := msgs[len(msgs)-1]
	if !strings.Contains(last.Content, "Attached media") {
		t.Fatalf("expected attached media section, got: %s", last.Content)
	}
	if !strings.Contains(last.Content, "a.png") || !strings.Contains(last.Content, "b.txt") {
		t.Fatalf("expected media names included, got: %s", last.Content)
	}
}
