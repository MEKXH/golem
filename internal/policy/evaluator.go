package policy

import "strings"

// Evaluator performs pure policy decisions.
type Evaluator struct {
	mode            Mode
	requireApproval map[string]struct{}
}

// NewEvaluator builds a deterministic, side-effect free evaluator.
func NewEvaluator(cfg Config) Evaluator {
	requireApproval := make(map[string]struct{}, len(cfg.RequireApproval))
	for _, toolName := range cfg.RequireApproval {
		normalized := normalizeToolName(toolName)
		if normalized == "" {
			continue
		}
		requireApproval[normalized] = struct{}{}
	}

	return Evaluator{
		mode:            normalizeMode(cfg.Mode),
		requireApproval: requireApproval,
	}
}

// Evaluate returns a deterministic decision for the given input.
func (e Evaluator) Evaluate(input Input) Decision {
	toolName := normalizeToolName(input.ToolName)

	if decision, matched := e.evaluateFutureRules(toolName); matched {
		return decision
	}

	switch e.mode {
	case ModeOff:
		return Decision{Action: ActionAllow}
	case ModeRelaxed:
		return Decision{Action: ActionAllow}
	case ModeStrict:
		if _, ok := e.requireApproval[toolName]; ok {
			return Decision{Action: ActionRequireApproval}
		}
		return Decision{Action: ActionAllow}
	default:
		return Decision{Action: ActionDeny, Reason: "unknown policy mode"}
	}
}

func normalizeMode(mode Mode) Mode {
	switch strings.ToLower(strings.TrimSpace(string(mode))) {
	case string(ModeStrict):
		return ModeStrict
	case string(ModeRelaxed):
		return ModeRelaxed
	case string(ModeOff):
		return ModeOff
	default:
		return Mode(strings.ToLower(strings.TrimSpace(string(mode))))
	}
}

func normalizeToolName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (e Evaluator) evaluateFutureRules(_ string) (Decision, bool) {
	return Decision{}, false
}
