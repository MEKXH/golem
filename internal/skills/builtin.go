package skills

import (
	"os"
	"path/filepath"
)

// DefaultBuiltinSkills returns the built-in skills shipped with golem.
func DefaultBuiltinSkills() map[string]string {
	return map[string]string{
		"weather": `---
name: weather
description: "Get current weather and short forecasts without an API key."
---

# Weather

Use ` + "`curl`" + ` with public services.

- Current: ` + "`curl -s \"wttr.in/Beijing?format=3\"`" + `
- Compact: ` + "`curl -s \"wttr.in/New+York?format=%l:+%c+%t\"`" + `
- Forecast: ` + "`curl -s \"wttr.in/Tokyo?T\"`" + `
`,
		"summarize": `---
name: summarize
description: "Summarize content from URLs or local files using concise outputs."
---

# Summarize

When users ask "summarize this", "what is this link about", or "extract key points":

1. Fetch or open the source content.
2. Return a short, structured summary.
3. Ask whether the user wants deeper sections.
`,
		"github": `---
name: github
description: "Interact with GitHub via gh CLI for issues, PRs, and workflow runs."
---

# GitHub

Prefer ` + "`gh`" + ` commands for repository operations.

- PR checks: ` + "`gh pr checks <number> --repo owner/repo`" + `
- List runs: ` + "`gh run list --repo owner/repo --limit 10`" + `
- Issue list: ` + "`gh issue list --repo owner/repo`" + `
`,
		"tmux": `---
name: tmux
description: "Control tmux sessions when interactive TTY workflows are required."
---

# tmux

Use tmux only for interactive tasks that cannot be done with plain shell execution.

- Create: ` + "`tmux new -d -s golem`" + `
- Send keys: ` + "`tmux send-keys -t golem 'echo hello' Enter`" + `
- Capture: ` + "`tmux capture-pane -p -t golem -S -200`" + `
`,
		"skill-creator": `---
name: skill-creator
description: "Create or update SKILL.md based skills with clear trigger descriptions."
---

# Skill Creator

When defining a skill:

1. Write exact trigger conditions in frontmatter description.
2. Keep instructions procedural and concise.
3. Add scripts/references only when reusable.
`,
	}
}

// EnsureBuiltinSkills writes default builtin skills into <configDir>/builtin-skills
// when they do not already exist.
func EnsureBuiltinSkills(configDir string) error {
	builtinRoot := filepath.Join(configDir, "builtin-skills")
	if err := os.MkdirAll(builtinRoot, 0755); err != nil {
		return err
	}

	for name, content := range DefaultBuiltinSkills() {
		dir := filepath.Join(builtinRoot, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		path := filepath.Join(dir, "SKILL.md")
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}
