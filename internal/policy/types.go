package policy

// Action is the policy decision for a tool execution request.
type Action string

const (
	ActionAllow           Action = "allow"
	ActionDeny            Action = "deny"
	ActionRequireApproval Action = "require_approval"
)

// Mode controls evaluator behavior.
type Mode string

const (
	ModeStrict  Mode = "strict"
	ModeRelaxed Mode = "relaxed"
	ModeOff     Mode = "off"
)

// Config contains policy settings required by the evaluator.
type Config struct {
	Mode            Mode
	RequireApproval []string
}

// Input is the minimum evaluation context.
type Input struct {
	ToolName string
}

// Decision is the deterministic policy result.
type Decision struct {
	Action Action
	Reason string
}
