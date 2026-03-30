package render

import (
	"testing"
)

func TestSplitThink_WithThinkBlock(t *testing.T) {
	input := "<think>I need to consider this carefully.</think>Here is my response."

	think, response, found := SplitThink(input)

	if !found {
		t.Fatal("expected found=true, got false")
	}
	if think != "I need to consider this carefully." {
		t.Errorf("expected think='I need to consider this carefully.', got %q", think)
	}
	if response != "Here is my response." {
		t.Errorf("expected response='Here is my response.', got %q", response)
	}
}

func TestSplitThink_NoThinkBlock(t *testing.T) {
	input := "Just a regular response with no thinking."

	think, response, found := SplitThink(input)

	if found {
		t.Fatal("expected found=false, got true")
	}
	if think != "" {
		t.Errorf("expected think='', got %q", think)
	}
	if response != input {
		t.Errorf("expected response=%q, got %q", input, response)
	}
}

func TestSplitThink_EmptyThinkBlock(t *testing.T) {
	input := "<think></think>text after empty think"

	think, response, found := SplitThink(input)

	if !found {
		t.Fatal("expected found=true for empty think block, got false")
	}
	if think != "" {
		t.Errorf("expected think='' (trimmed empty), got %q", think)
	}
	if response != "text after empty think" {
		t.Errorf("expected response='text after empty think', got %q", response)
	}
}

func TestSplitThink_MultilineThink(t *testing.T) {
	input := `<think>
First, I should analyze the problem.
Then, I should consider edge cases.
Finally, I will formulate my answer.
</think>Here is the final answer.`

	think, response, found := SplitThink(input)

	if !found {
		t.Fatal("expected found=true, got false")
	}

	// Think content should be trimmed
	expectedThink := "First, I should analyze the problem.\nThen, I should consider edge cases.\nFinally, I will formulate my answer."
	if think != expectedThink {
		t.Errorf("expected think=%q, got %q", expectedThink, think)
	}
	if response != "Here is the final answer." {
		t.Errorf("expected response='Here is the final answer.', got %q", response)
	}
}

func TestSplitThink_UnclosedThinkBlock(t *testing.T) {
	input := "Before think <think>inside unclosed think block"

	think, response, found := SplitThink(input)

	if !found {
		t.Fatal("expected found=true for unclosed think block, got false")
	}
	if think != "inside unclosed think block" {
		t.Errorf("expected think='inside unclosed think block', got %q", think)
	}
	if response != "Before think" {
		t.Errorf("expected response='Before think', got %q", response)
	}
}

func TestSplitThink_OnlyThinkStart(t *testing.T) {
	input := "<think>"

	think, response, found := SplitThink(input)

	if !found {
		t.Fatal("expected found=true for unclosed think block, got false")
	}
	if think != "" {
		t.Errorf("expected think='', got %q", think)
	}
	if response != "" {
		t.Errorf("expected response='', got %q", response)
	}
}

func TestSplitThink_StartAfterText(t *testing.T) {
	input := "Regular text here<think>And thinking here</think>And some more text here"

	think, response, found := SplitThink(input)

	if !found {
		t.Fatal("expected found=true, got false")
	}
	if think != "And thinking here" {
		t.Errorf("expected think='And thinking here', got %q", think)
	}
	if response != "Regular text hereAnd some more text here" {
		t.Errorf("expected response='Regular text hereAnd some more text here', got %q", response)
	}
}
