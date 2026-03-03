package approval

import "time"

// RequestStatus 表示审批请求在生命周期中的不同状态。
type RequestStatus string

const (
	StatusPending  RequestStatus = "pending"  // 待处理：请求已发起，正等待人工决策
	StatusApproved RequestStatus = "approved" // 已通过：人工已批准该工具执行
	StatusRejected RequestStatus = "rejected" // 已拒绝：人工已显式拒绝该工具执行
	StatusExpired  RequestStatus = "expired"  // 已过期：请求在有效期（TTL）内未获得决策
)

// Request 表示一条持久化的审批请求记录。
type Request struct {
	ID           string        `json:"id"`                      // 唯一的请求 ID
	ToolName     string        `json:"tool_name"`               // 待执行工具的名称
	ArgsJSON     string        `json:"args_json"`               // 工具执行的参数（JSON 格式）
	Reason       string        `json:"reason,omitempty"`        // 触发审批的原因
	DecisionNote string        `json:"decision_note,omitempty"` // 审批决策时的备注说明
	Status       RequestStatus `json:"status"`                  // 当前审批状态
	RequestedAt  time.Time     `json:"requested_at"`            // 请求发起时间
	ExpiresAt    time.Time     `json:"expires_at,omitempty"`    // 请求过期时间
	DecidedAt    time.Time     `json:"decided_at,omitempty"`    // 决策完成时间
	DecidedBy    string        `json:"decided_by,omitempty"`    // 决策者标识（用户名或系统）
}

// CreateInput 包含了创建新审批请求所需的输入字段。
type CreateInput struct {
	ToolName string        // 工具名称
	ArgsJSON string        // 参数 JSON
	Reason   string        // 申请原因
	TTL      time.Duration // 有效期时长
}

// DecisionInput 包含了对请求进行批准或拒绝所需的输入字段。
type DecisionInput struct {
	DecidedBy string // 决策者
	Note      string // 决策备注
}

// Query 包含了用于列表过滤的查询字段。
type Query struct {
	ID       string        // 按 ID 过滤
	Status   RequestStatus // 按状态过滤
	ToolName string        // 按工具名过滤
}
