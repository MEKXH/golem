package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type fakeRenderer struct {
	inputs []string
}

func (f *fakeRenderer) Render(s string) (string, error) {
	f.inputs = append(f.inputs, s)
	return "R:" + s, nil
}

func TestRenderResponseParts_WithThinkRendersBoth(t *testing.T) {
	r := &fakeRenderer{}
	think, main, hasThink := renderResponseParts("<think>**t**</think>**m**", r)
	if !hasThink {
		t.Fatal("expected hasThink=true")
	}
	if len(r.inputs) != 2 {
		t.Fatalf("expected 2 renders, got %d", len(r.inputs))
	}
	if think != "R:**t**" {
		t.Fatalf("unexpected think: %s", think)
	}
	if main != "R:**m**" {
		t.Fatalf("unexpected main: %s", main)
	}
}

func TestRenderResponseParts_NoThinkRendersMainOnly(t *testing.T) {
	r := &fakeRenderer{}
	think, main, hasThink := renderResponseParts("**m**", r)
	if hasThink {
		t.Fatal("expected hasThink=false")
	}
	if len(r.inputs) != 1 {
		t.Fatalf("expected 1 render, got %d", len(r.inputs))
	}
	if think != "" {
		t.Fatalf("expected empty think, got %s", think)
	}
	if main != "R:**m**" {
		t.Fatalf("unexpected main: %s", main)
	}
}

func TestRenderResponseParts_ThinkAndMainAreRendered(t *testing.T) {
	r := &fakeRenderer{}
	think, main, hasThink := renderResponseParts("<think>t</think>m", r)
	if !hasThink || think == "" || main == "" {
		t.Fatal("expected rendered think and main")
	}
}

func TestView_RendersFooter(t *testing.T) {
	m := model{
		textarea: textarea.New(),
		viewport: viewport.New(10, 10),
		spinner:  spinner.New(),
		thinking: false,
	}

	output := m.View()

	expected := []string{"Enter", "Send", "•", "/new", "Reset", "•", "Esc", "Quit"}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected view to contain %q, but it didn't. Output:\n%s", exp, output)
		}
	}
}

func TestRenderMessage_ToolIcons(t *testing.T) {
	// Force color output for testing
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := model{
		width: 100, // Ensure width is enough
	}
	msg := &ChatMessage{
		Role:    "golem",
		Content: "test",
		Tools: []ToolLog{
			{Name: "success_tool", Result: "ok"},
			{Name: "error_tool", Err: fmt.Errorf("fail")},
		},
	}

	output := m.renderMessage(msg)

	// We expect ANSI escape sequences for colors.
	// Green for success
	// Red for error
	if !strings.Contains(output, "✔") {
		t.Error("expected output to contain ✔")
	}
	if !strings.Contains(output, "✖") {
		t.Error("expected output to contain ✖")
	}
	// Check if color codes are present (basic check for escape char)
	if !strings.Contains(output, "\x1b[") {
		t.Error("expected output to contain ANSI escape codes")
	}
}

func TestRenderMessage_ErrorState(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	m := model{width: 100}

	msg := &ChatMessage{
		Role:    "golem",
		Content: "error occurred",
		IsError: true,
	}

	output := m.renderMessage(msg)

	if !strings.Contains(output, "ERROR") {
		t.Error("expected output to contain ERROR label")
	}
	if strings.Contains(output, "GOLEM") {
		t.Error("expected output not to contain GOLEM label for error messages")
	}
}
