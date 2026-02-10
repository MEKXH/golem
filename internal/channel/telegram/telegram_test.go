package telegram

import (
	"strings"
	"testing"
)

func TestMarkdownToHTML_RendersBoldAndCode(t *testing.T) {
	out := markdownToHTML("**b** `c`")
	if strings.Contains(out, "&lt;b&gt;") {
		t.Fatalf("expected bold tags to be real HTML, got: %s", out)
	}
	if !strings.Contains(out, "<b>b</b>") {
		t.Fatalf("expected bold to render, got: %s", out)
	}
	if !strings.Contains(out, "<code>c</code>") {
		t.Fatalf("expected code to render, got: %s", out)
	}
}

func TestRenderMessageHTML_IncludesThinkContent(t *testing.T) {
	out := renderMessageHTML("<think>**t**</think>**m**")
	if strings.Contains(out, "<think>") {
		t.Fatalf("expected think tags removed, got: %s", out)
	}
	if !strings.Contains(out, "Thinking:") {
		t.Fatalf("expected thinking label, got: %s", out)
	}
	if !strings.Contains(out, "<b>t</b>") || !strings.Contains(out, "<b>m</b>") {
		t.Fatalf("expected rendered think and main, got: %s", out)
	}
}

func TestParseInt64_Valid(t *testing.T) {
	got, err := parseInt64("12345")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != 12345 {
		t.Fatalf("expected 12345, got %d", got)
	}
}

func TestParseInt64_Invalid(t *testing.T) {
	if _, err := parseInt64("not-a-number"); err == nil {
		t.Fatal("expected error for invalid chat id")
	}
}
