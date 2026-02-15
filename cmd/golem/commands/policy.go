package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/audit"
	"github.com/MEKXH/golem/internal/config"
	"github.com/spf13/cobra"
)

func NewPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage runtime policy mode",
	}

	cmd.AddCommand(
		newPolicyOffCmd(),
		newPolicyStrictCmd(),
		newPolicyRelaxedCmd(),
		newPolicyStatusCmd(),
	)

	return cmd
}

func newPolicyOffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "off",
		Short: "Temporarily disable policy checks (TTL required)",
		RunE:  runPolicyOff,
	}
	cmd.Flags().String("ttl", "", "TTL duration, for example 15m, 1h")
	return cmd
}

func newPolicyStrictCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "strict",
		Short: "Set policy mode to strict",
		RunE:  runPolicyStrict,
	}
}

func newPolicyRelaxedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "relaxed",
		Short: "Set policy mode to relaxed",
		RunE:  runPolicyRelaxed,
	}
}

func newPolicyStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current policy mode and risk hint",
		RunE:  runPolicyStatus,
	}
}

func runPolicyOff(cmd *cobra.Command, args []string) error {
	ttlRaw, _ := cmd.Flags().GetString("ttl")
	ttlRaw = strings.TrimSpace(ttlRaw)
	if ttlRaw == "" {
		return fmt.Errorf("--ttl is required for policy off mode")
	}
	ttl, err := time.ParseDuration(ttlRaw)
	if err != nil || ttl <= 0 {
		return fmt.Errorf("invalid --ttl duration: %q", ttlRaw)
	}

	cfg, workspacePath, err := loadPolicyConfig()
	if err != nil {
		return err
	}

	cfg.Policy.Mode = "off"
	cfg.Policy.OffTTL = ttl.String()
	cfg.Policy.AllowPersistentOff = false
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	appendPolicySwitchAudit(workspacePath, cfg)
	fmt.Printf("Policy set to off (ttl=%s).\n", cfg.Policy.OffTTL)
	return nil
}

func runPolicyStrict(cmd *cobra.Command, args []string) error {
	cfg, workspacePath, err := loadPolicyConfig()
	if err != nil {
		return err
	}

	cfg.Policy.Mode = "strict"
	cfg.Policy.OffTTL = ""
	cfg.Policy.AllowPersistentOff = false
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	appendPolicySwitchAudit(workspacePath, cfg)
	fmt.Println("Policy set to strict.")
	return nil
}

func runPolicyRelaxed(cmd *cobra.Command, args []string) error {
	cfg, workspacePath, err := loadPolicyConfig()
	if err != nil {
		return err
	}

	cfg.Policy.Mode = "relaxed"
	cfg.Policy.OffTTL = ""
	cfg.Policy.AllowPersistentOff = false
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	appendPolicySwitchAudit(workspacePath, cfg)
	fmt.Println("Policy set to relaxed.")
	return nil
}

func runPolicyStatus(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadPolicyConfig()
	if err != nil {
		return err
	}

	mode := strings.TrimSpace(cfg.Policy.Mode)
	offTTL := strings.TrimSpace(cfg.Policy.OffTTL)
	if offTTL == "" {
		offTTL = "none"
	}
	approvalList := "none"
	if len(cfg.Policy.RequireApproval) > 0 {
		approvalList = strings.Join(cfg.Policy.RequireApproval, ", ")
	}

	fmt.Println("Policy status:")
	fmt.Printf("  mode: %s\n", mode)
	fmt.Printf("  off_ttl: %s\n", offTTL)
	fmt.Printf("  allow_persistent_off: %t\n", cfg.Policy.AllowPersistentOff)
	fmt.Printf("  require_approval: %s\n", approvalList)

	if warning, ok := persistentOffWarningMessage(cfg.Policy.Mode, cfg.Policy.OffTTL); ok {
		fmt.Printf("  risk: HIGH-RISK (%s)\n", warning)
	} else if strings.EqualFold(mode, "off") {
		fmt.Println("  risk: limited (ttl-based off mode)")
	} else {
		fmt.Println("  risk: normal")
	}

	return nil
}

func loadPolicyConfig() (*config.Config, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config: %w", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return nil, "", fmt.Errorf("invalid workspace: %w", err)
	}
	return cfg, workspacePath, nil
}

func appendPolicySwitchAudit(workspacePath string, cfg *config.Config) {
	if cfg == nil || strings.TrimSpace(workspacePath) == "" {
		return
	}
	mode := strings.TrimSpace(cfg.Policy.Mode)
	offTTL := strings.TrimSpace(cfg.Policy.OffTTL)
	if offTTL == "" {
		offTTL = "none"
	}
	requireApproval := "-"
	if len(cfg.Policy.RequireApproval) > 0 {
		requireApproval = strings.Join(cfg.Policy.RequireApproval, ",")
	}

	evt := audit.Event{
		Time: time.Now().UTC(),
		Type: "policy_cli_switch",
		Result: fmt.Sprintf(
			"mode=%s off_ttl=%s allow_persistent_off=%t require_approval=%s",
			mode,
			offTTL,
			cfg.Policy.AllowPersistentOff,
			requireApproval,
		),
	}
	_ = audit.NewWriter(workspacePath).Append(evt)
}
