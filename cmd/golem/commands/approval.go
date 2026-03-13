package commands

import (
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/approval"
	"github.com/MEKXH/golem/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewApprovalCmd 创建审批管理命令。
func NewApprovalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approval",
		Short: "Manage approval requests",
	}

	cmd.AddCommand(
		newApprovalListCmd(),
		newApprovalApproveCmd(),
		newApprovalRejectCmd(),
	)

	return cmd
}

func newApprovalListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List pending approval requests",
		RunE:  runApprovalList,
	}
}

func newApprovalApproveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve an approval request",
		Args:  cobra.ExactArgs(1),
		RunE:  runApprovalApprove,
	}
	cmd.Flags().String("by", "", "Decision maker")
	cmd.Flags().String("note", "", "Decision note")
	_ = cmd.MarkFlagRequired("by")
	return cmd
}

func newApprovalRejectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reject <id>",
		Short: "Reject an approval request",
		Args:  cobra.ExactArgs(1),
		RunE:  runApprovalReject,
	}
	cmd.Flags().String("by", "", "Decision maker")
	cmd.Flags().String("note", "", "Decision note")
	_ = cmd.MarkFlagRequired("by")
	return cmd
}

func runApprovalList(cmd *cobra.Command, args []string) error {
	svc, err := loadApprovalService()
	if err != nil {
		return err
	}

	requests, err := svc.List(approval.Query{Status: approval.StatusPending})
	if err != nil {
		return err
	}
	if len(requests) == 0 {
		fmt.Println("No pending approvals. Pending requests will appear here when an agent proposes restricted actions.")
		return nil
	}

	var (
		wID     = 20
		wTool   = 15
		wReason = 40
		wStatus = 10

		colHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8E4EC6")). // Purple
				Bold(true).
				MarginRight(1)

		idStyleBase = lipgloss.NewStyle().
				Width(wID).
				MarginRight(1)

		toolStyleBase = lipgloss.NewStyle().
				Width(wTool).
				MarginRight(1)

		reasonStyleBase = lipgloss.NewStyle().
				Width(wReason).
				MarginRight(1)

		statusStyleBase = lipgloss.NewStyle().
				Width(wStatus).
				MarginRight(1)

		pendingColor = lipgloss.Color("#FFA500") // Orange
	)

	fmt.Println("Pending Approvals:")
	fmt.Println()

	// Render Headers
	headers := lipgloss.JoinHorizontal(lipgloss.Top,
		colHeaderStyle.Width(wID).Render("ID"),
		colHeaderStyle.Width(wTool).Render("TOOL"),
		colHeaderStyle.Width(wStatus).Render("STATUS"),
		colHeaderStyle.Width(wReason).Render("REASON"),
	)
	fmt.Printf("  %s\n", headers)

	// Render Separator
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginRight(1)
	separator := lipgloss.JoinHorizontal(lipgloss.Top,
		sepStyle.Render(strings.Repeat("─", wID)),
		sepStyle.Render(strings.Repeat("─", wTool)),
		sepStyle.Render(strings.Repeat("─", wStatus)),
		sepStyle.Render(strings.Repeat("─", wReason)),
	)
	fmt.Printf("  %s\n", separator)

	for _, req := range requests {
		reason := req.Reason
		if reason == "" {
			reason = "-"
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			idStyleBase.Render(truncate(req.ID, wID)),
			toolStyleBase.Render(truncate(req.ToolName, wTool)),
			statusStyleBase.Foreground(pendingColor).Render(string(req.Status)),
			reasonStyleBase.Render(truncate(reason, wReason)),
		)
		fmt.Printf("  %s\n", row)
	}

	return nil
}

func runApprovalApprove(cmd *cobra.Command, args []string) error {
	return runApprovalDecision(cmd, args[0], true)
}

func runApprovalReject(cmd *cobra.Command, args []string) error {
	return runApprovalDecision(cmd, args[0], false)
}

func runApprovalDecision(cmd *cobra.Command, id string, approve bool) error {
	svc, err := loadApprovalService()
	if err != nil {
		return err
	}

	by, _ := cmd.Flags().GetString("by")
	note, _ := cmd.Flags().GetString("note")
	if strings.TrimSpace(by) == "" {
		return fmt.Errorf("--by is required")
	}

	decision := approval.DecisionInput{
		DecidedBy: strings.TrimSpace(by),
		Note:      strings.TrimSpace(note),
	}

	if approve {
		if _, err := svc.Approve(id, decision); err != nil {
			return err
		}
		fmt.Printf("Approval %s approved.\n", id)
		return nil
	}

	if _, err := svc.Reject(id, decision); err != nil {
		return err
	}
	fmt.Printf("Approval %s rejected.\n", id)
	return nil
}

func loadApprovalService() (*approval.Service, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return nil, fmt.Errorf("invalid workspace: %w", err)
	}
	return approval.NewService(workspacePath), nil
}
