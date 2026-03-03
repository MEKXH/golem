package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/skills"
)

// SkillsCommand 实现 /skills 命令 — 用于查看已安装技能及其详细信息。
// 使用方式:
//
//	/skills [list] - 列出所有已发现的技能
//	/skills show <name> - 显示指定技能的完整 Markdown 内容
type SkillsCommand struct{}

// Name 返回命令名称。
func (c *SkillsCommand) Name() string { return "skills" }

// Description 返回命令描述。
func (c *SkillsCommand) Description() string { return "Manage skills (list|show <name>)" }

// Execute 执行技能管理逻辑。
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
