package policy

// Action 是工具执行请求的策略决策。
type Action string

const (
	ActionAllow           Action = "allow"
	ActionDeny            Action = "deny"
	ActionRequireApproval Action = "require_approval"
)

// Mode 控制评估器的行为。
type Mode string

const (
	ModeStrict  Mode = "strict"
	ModeRelaxed Mode = "relaxed"
	ModeOff     Mode = "off"
)

// Config 包含评估器所需的策略设置。
type Config struct {
	Mode            Mode
	RequireApproval []string
}

// Input 是最小的评估上下文。
type Input struct {
	ToolName string
}

// Decision 是确定性的策略结果。
type Decision struct {
	Action Action
	Reason string
}
