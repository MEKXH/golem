// Package policy 实现 Golem 的运行时安全策略评估引擎。
// 它根据配置的模式（严格、宽松、关闭）对工具执行请求进行准入控制。
package policy

import "strings"

// Evaluator 负责执行纯粹的策略决策逻辑，不包含副作用。
type Evaluator struct {
	mode            Mode                // 当前运行模式
	requireApproval map[string]struct{} // 需要人工审批的工具名称映射表
}

// NewEvaluator 根据提供的配置构建一个确定性的评估器实例。
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

// Evaluate 根据当前策略模式和输入上下文（如工具名称）返回一个确定性的准入决策。
func (e Evaluator) Evaluate(input Input) Decision {
	toolName := normalizeToolName(input.ToolName)

	// 预留：评估未来可能的动态规则
	if decision, matched := e.evaluateFutureRules(toolName); matched {
		return decision
	}

	switch e.mode {
	case ModeOff:
		// 策略关闭模式：允许所有操作
		return Decision{Action: ActionAllow}
	case ModeRelaxed:
		// 宽松模式：目前默认允许所有操作，未来可能增加敏感操作审计
		return Decision{Action: ActionAllow}
	case ModeStrict:
		// 严格模式：检查工具是否在审批名单中
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
