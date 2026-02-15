package commands

import (
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/approval"
	"github.com/MEKXH/golem/internal/config"
	"github.com/spf13/cobra"
)

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
		fmt.Println("No pending approvals.")
		return nil
	}

	for _, req := range requests {
		fmt.Printf("%s %s %s\n", req.ID, req.ToolName, req.Status)
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
