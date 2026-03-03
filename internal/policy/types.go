package policy

// Action 表示策略评估后针对工具执行请求的决策动作。
type Action string

const (
	ActionAllow           Action = "allow"            // 允许执行
	ActionDeny            Action = "deny"             // 拒绝执行
	ActionRequireApproval Action = "require_approval" // 需要人工审批后执行
)

// Mode 控制策略评估器的运行模式。
type Mode string

const (
	ModeStrict  Mode = "strict"  // 严格模式：不在允许列表中的敏感操作需审批
	ModeRelaxed Mode = "relaxed" // 宽松模式：目前允许所有操作，未来可能增加风险预警
	ModeOff     Mode = "off"     // 关闭模式：不进行任何策略拦截
)

// Config 包含评估器初始化所需的策略设置。
type Config struct {
	Mode            Mode     // 运行模式
	RequireApproval []string // 需要审批的工具列表
}

// Input 包含单次策略评估所需的上下文信息。
type Input struct {
	ToolName string // 待执行的工具名称
}

// Decision 封装了策略评估的最终确定性结果。
type Decision struct {
	Action Action // 决策动作
	Reason string // 决策原因（特别是在拒绝时）
}
