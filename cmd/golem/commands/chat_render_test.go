package commands

import "testing"

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
