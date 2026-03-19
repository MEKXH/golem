package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/skills"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewSkillsCmd 创建技能管理子命令。
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
		newSkillsSearchCmd(),
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
	var yes bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed skill",
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsRemove,
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func newSkillsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show skill content",
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsShow,
	}
}

func newSkillsSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [keyword]",
		Short: "Search available remote skills",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runSkillsSearch,
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
		fmt.Println("No skills installed. Use 'golem skills install <owner/repo>' to add one.")
		return nil
	}

	wName := 20
	wSource := 12
	wDesc := 30

	colHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8E4EC6")). // Purple
		Bold(true).
		MarginRight(1)

	nameStyle := lipgloss.NewStyle().Width(wName).MarginRight(1)
	sourceStyle := lipgloss.NewStyle().Width(wSource).MarginRight(1)
	descStyle := lipgloss.NewStyle().Width(wDesc)

	headers := lipgloss.JoinHorizontal(lipgloss.Top,
		colHeaderStyle.Width(wName).Render("NAME"),
		colHeaderStyle.Width(wSource).Render("SOURCE"),
		colHeaderStyle.Width(wDesc).Render("DESCRIPTION"),
	)
	fmt.Printf("  %s\n", headers)

	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginRight(1)
	separator := lipgloss.JoinHorizontal(lipgloss.Top,
		sepStyle.Render(strings.Repeat("─", wName)),
		sepStyle.Render(strings.Repeat("─", wSource)),
		sepStyle.Render(strings.Repeat("─", wDesc)),
	)
	fmt.Printf("  %s\n", separator)

	for _, s := range skillList {
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			nameStyle.Render(truncate(s.Name, wName)),
			sourceStyle.Render(s.Source),
			descStyle.Render(truncate(s.Description, 50)),
		)
		fmt.Printf("  %s\n", row)
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
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Printf("Are you sure you want to remove skill '%s'? [y/N] ", args[0])
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Skill removal cancelled.")
			return nil
		}
	}

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

func runSkillsSearch(cmd *cobra.Command, args []string) error {
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

	list, err := installer.Search(ctx)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	keyword := ""
	if len(args) > 0 {
		keyword = strings.ToLower(strings.TrimSpace(args[0]))
	}

	filtered := list[:0]
	for _, item := range list {
		if keyword == "" {
			filtered = append(filtered, item)
			continue
		}

		haystack := strings.ToLower(item.Name + " " + item.Repository + " " + item.Description + " " + strings.Join(item.Tags, " "))
		if strings.Contains(haystack, keyword) {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) == 0 {
		fmt.Println("No matching skills found. Try another search term, or use 'golem skills search' to list all available skills.")
		return nil
	}

	wName := 20
	wRepo := 32
	wDesc := 30

	colHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8E4EC6")). // Purple
		Bold(true).
		MarginRight(1)

	nameStyle := lipgloss.NewStyle().Width(wName).MarginRight(1)
	repoStyle := lipgloss.NewStyle().Width(wRepo).MarginRight(1)
	descStyle := lipgloss.NewStyle().Width(wDesc)

	headers := lipgloss.JoinHorizontal(lipgloss.Top,
		colHeaderStyle.Width(wName).Render("NAME"),
		colHeaderStyle.Width(wRepo).Render("REPOSITORY"),
		colHeaderStyle.Width(wDesc).Render("DESCRIPTION"),
	)
	fmt.Printf("  %s\n", headers)

	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginRight(1)
	separator := lipgloss.JoinHorizontal(lipgloss.Top,
		sepStyle.Render(strings.Repeat("─", wName)),
		sepStyle.Render(strings.Repeat("─", wRepo)),
		sepStyle.Render(strings.Repeat("─", wDesc)),
	)
	fmt.Printf("  %s\n", separator)

	for _, item := range filtered {
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			nameStyle.Render(truncate(item.Name, wName)),
			repoStyle.Render(truncate(item.Repository, wRepo)),
			descStyle.Render(truncate(item.Description, 50)),
		)
		fmt.Printf("  %s\n", row)
	}

	return nil
}
