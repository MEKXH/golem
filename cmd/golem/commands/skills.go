package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/skills"
	"github.com/spf13/cobra"
)

func NewSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage installed skills",
	}

	cmd.AddCommand(
		newSkillsListCmd(),
		newSkillsInstallCmd(),
		newSkillsRemoveCmd(),
		newSkillsShowCmd(),
	)

	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed skills",
		RunE:  runSkillsList,
	}
}

func newSkillsInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <owner/repo>",
		Short: "Install a skill from GitHub",
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsInstall,
	}
}

func newSkillsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed skill",
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsRemove,
	}
}

func newSkillsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show skill content",
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsShow,
	}
}

func loadSkillsLoader() (*skills.Loader, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return nil, fmt.Errorf("invalid workspace: %w", err)
	}
	return skills.NewLoader(workspacePath), nil
}

func runSkillsList(cmd *cobra.Command, args []string) error {
	loader, err := loadSkillsLoader()
	if err != nil {
		return err
	}

	skillList := loader.ListSkills()
	if len(skillList) == 0 {
		fmt.Println("No skills installed.")
		return nil
	}

	fmt.Printf("  %-20s %-12s %s\n", "NAME", "SOURCE", "DESCRIPTION")
	fmt.Printf("  %-20s %-12s %s\n",
		strings.Repeat("-", 20),
		strings.Repeat("-", 12),
		strings.Repeat("-", 30),
	)

	for _, s := range skillList {
		fmt.Printf("  %-20s %-12s %s\n",
			truncate(s.Name, 20),
			s.Source,
			truncate(s.Description, 50),
		)
	}

	return nil
}

func runSkillsInstall(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return fmt.Errorf("invalid workspace: %w", err)
	}

	installer := skills.NewInstaller(workspacePath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Installing skill from %s...\n", args[0])
	if err := installer.Install(ctx, args[0]); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	fmt.Println("Skill installed successfully.")
	return nil
}

func runSkillsRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return fmt.Errorf("invalid workspace: %w", err)
	}

	installer := skills.NewInstaller(workspacePath)
	if err := installer.Uninstall(args[0]); err != nil {
		return err
	}

	fmt.Printf("Skill '%s' removed.\n", args[0])
	return nil
}

func runSkillsShow(cmd *cobra.Command, args []string) error {
	loader, err := loadSkillsLoader()
	if err != nil {
		return err
	}

	content, err := loader.LoadSkill(args[0])
	if err != nil {
		return err
	}

	fmt.Println(content)
	return nil
}
