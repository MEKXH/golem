package approval

import "time"

// RequestStatus is the lifecycle state of an approval request.
type RequestStatus string

const (
	StatusPending  RequestStatus = "pending"
	StatusApproved RequestStatus = "approved"
	StatusRejected RequestStatus = "rejected"
	StatusExpired  RequestStatus = "expired"
)

// Request is a persisted approval request record.
type Request struct {
	ID           string        `json:"id"`
	ToolName     string        `json:"tool_name"`
	ArgsJSON     string        `json:"args_json"`
	Reason       string        `json:"reason,omitempty"`
	DecisionNote string        `json:"decision_note,omitempty"`
	Status       RequestStatus `json:"status"`
	RequestedAt  time.Time     `json:"requested_at"`
	ExpiresAt    time.Time     `json:"expires_at,omitempty"`
	DecidedAt    time.Time     `json:"decided_at,omitempty"`
	DecidedBy    string        `json:"decided_by,omitempty"`
}

// CreateInput contains fields needed to create an approval request.
type CreateInput struct {
	ToolName string
	ArgsJSON string
	Reason   string
	TTL      time.Duration
}

// DecisionInput contains fields needed to approve/reject a request.
type DecisionInput struct {
	DecidedBy string
	Note      string
}

// Query filters approval requests when listing.
type Query struct {
	ID       string
	Status   RequestStatus
	ToolName string
}
