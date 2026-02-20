package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/skills"
)

// SkillsCommand implements /skills — list and inspect installed skills.
// Subcommands: list, show <name>
type SkillsCommand struct{}

func (c *SkillsCommand) Name() string        { return "skills" }
func (c *SkillsCommand) Description() string { return "Manage skills (list|show <name>)" }

func (c *SkillsCommand) Execute(_ context.Context, args string, env Env) Result {
	sub, rest, _ := strings.Cut(args, " ")
	sub = strings.ToLower(strings.TrimSpace(sub))
	rest = strings.TrimSpace(rest)

	loader := skills.NewLoader(env.WorkspacePath)

	switch sub {
	case "", "list":
		return skillsList(loader)
	case "show":
		return skillsShow(loader, rest)
	default:
		return Result{Content: "Usage: `/skills [list|show <name>]`"}
	}
}

func skillsList(loader *skills.Loader) Result {
	list := loader.ListSkills()
	if len(list) == 0 {
		return Result{Content: "No skills installed."}
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Skills (%d):**\n\n", len(list)))
	for _, s := range list {
		sb.WriteString(fmt.Sprintf("- **%s** `%s` — %s\n", s.Name, s.Source, s.Description))
	}
	return Result{Content: sb.String()}
}

func skillsShow(loader *skills.Loader, name string) Result {
	if name == "" {
		return Result{Content: "Usage: `/skills show <name>`"}
	}
	content, err := loader.LoadSkill(name)
	if err != nil {
		return Result{Content: fmt.Sprintf("Error: %v", err)}
	}
	if len(content) > 2000 {
		content = content[:2000] + "\n...(truncated)"
	}
	return Result{Content: content}
}
