package policy

import "testing"

func TestEvaluate_StrictRequiresApprovalForConfiguredTool(t *testing.T) {
	ev := NewEvaluator(Config{Mode: ModeStrict, RequireApproval: []string{"exec"}})
	d := ev.Evaluate(Input{ToolName: "exec"})

	if d.Action != ActionRequireApproval {
		t.Fatalf("expected %q, got %q", ActionRequireApproval, d.Action)
	}
}

func TestEvaluate_StrictAllowsToolNotInRequireApprovalList(t *testing.T) {
	ev := NewEvaluator(Config{Mode: ModeStrict, RequireApproval: []string{"exec"}})
	d := ev.Evaluate(Input{ToolName: "read_file"})

	if d.Action != ActionAllow {
		t.Fatalf("expected %q, got %q", ActionAllow, d.Action)
	}
}

func TestEvaluate_RelaxedAllowsByDefault(t *testing.T) {
	ev := NewEvaluator(Config{Mode: ModeRelaxed, RequireApproval: []string{"exec"}})
	d := ev.Evaluate(Input{ToolName: "exec"})

	if d.Action != ActionAllow {
		t.Fatalf("expected %q, got %q", ActionAllow, d.Action)
	}
}

func TestEvaluate_OffAllowsAll(t *testing.T) {
	ev := NewEvaluator(Config{Mode: ModeOff, RequireApproval: []string{"exec"}})
	d := ev.Evaluate(Input{ToolName: "exec"})

	if d.Action != ActionAllow {
		t.Fatalf("expected %q, got %q", ActionAllow, d.Action)
	}
}

func TestEvaluate_UnknownModeDenies(t *testing.T) {
	ev := NewEvaluator(Config{Mode: "unknown"})
	d := ev.Evaluate(Input{ToolName: "exec"})

	if d.Action != ActionDeny {
		t.Fatalf("expected %q, got %q", ActionDeny, d.Action)
	}
}

func TestEvaluate_StrictRequireApprovalListIsNormalized(t *testing.T) {
	ev := NewEvaluator(Config{Mode: ModeStrict, RequireApproval: []string{"  EXEC  "}})
	d := ev.Evaluate(Input{ToolName: "exec"})

	if d.Action != ActionRequireApproval {
		t.Fatalf("expected %q, got %q", ActionRequireApproval, d.Action)
	}
}

func TestEvaluate_InputToolNameIsNormalized(t *testing.T) {
	ev := NewEvaluator(Config{Mode: ModeStrict, RequireApproval: []string{"exec"}})
	d := ev.Evaluate(Input{ToolName: "  ExEc "})

	if d.Action != ActionRequireApproval {
		t.Fatalf("expected %q, got %q", ActionRequireApproval, d.Action)
	}
}
