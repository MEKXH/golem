package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SkillInfo describes a loaded skill.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Source      string `json:"source"` // "workspace" | "global" | "builtin"
}

// Loader discovers and loads skill files.
type Loader struct {
	workspaceSkills string // workspace/skills/
	globalSkills    string // ~/.golem/skills/
	builtinSkills   string // builtin skills shipped with golem
}

// NewLoader creates a skill loader for the given workspace.
func NewLoader(workspacePath string) *Loader {
	homeDir, _ := os.UserHomeDir()
	return &Loader{
		workspaceSkills: filepath.Join(workspacePath, "skills"),
		globalSkills:    filepath.Join(homeDir, ".golem", "skills"),
		builtinSkills:   resolveBuiltinSkillsDir(homeDir),
	}
}

// ListSkills returns all discovered skills (workspace > global > builtin).
func (l *Loader) ListSkills() []SkillInfo {
	seen := make(map[string]bool)
	var skills []SkillInfo

	// Workspace skills (highest priority)
	for _, s := range l.scanDir(l.workspaceSkills, "workspace") {
		seen[s.Name] = true
		skills = append(skills, s)
	}

	// Global skills
	for _, s := range l.scanDir(l.globalSkills, "global") {
		if !seen[s.Name] {
			seen[s.Name] = true
			skills = append(skills, s)
		}
	}

	// Builtin skills
	for _, s := range l.scanDir(l.builtinSkills, "builtin") {
		if !seen[s.Name] {
			skills = append(skills, s)
		}
	}

	return skills
}

// LoadSkill reads the content of a skill by name.
func (l *Loader) LoadSkill(name string) (string, error) {
	// Search workspace first, then global, then builtin.
	for _, dir := range []string{l.workspaceSkills, l.globalSkills, l.builtinSkills} {
		path := filepath.Join(dir, name, "SKILL.md")
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("skill not found: %s", name)
}

// BuildSkillsSummary returns a formatted summary of all skills for system prompt injection.
func (l *Loader) BuildSkillsSummary() string {
	skills := l.ListSkills()
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Installed Skills\n\n")
	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", s.Name, s.Description))
	}
	return sb.String()
}

func (l *Loader) scanDir(dir, source string) []SkillInfo {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var skills []SkillInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}

		name, desc := parseSkillFrontmatter(entry.Name(), string(data))
		skills = append(skills, SkillInfo{
			Name:        name,
			Description: desc,
			Path:        skillPath,
			Source:      source,
		})
	}
	return skills
}

// parseSkillFrontmatter extracts name and description from YAML frontmatter.
// Expected format:
//
//	---
//	name: weather
//	description: "Query weather info"
//	---
func parseSkillFrontmatter(dirName, content string) (name, description string) {
	name = dirName
	description = "(no description)"

	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return
	}

	end := strings.Index(content[3:], "---")
	if end < 0 {
		return
	}

	frontmatter := content[3 : 3+end]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			name = strings.Trim(name, `"'`)
		}
		if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			description = strings.Trim(description, `"'`)
		}
	}
	return
}

func resolveBuiltinSkillsDir(homeDir string) string {
	if fromEnv := strings.TrimSpace(os.Getenv("GOLEM_BUILTIN_SKILLS_DIR")); fromEnv != "" {
		return fromEnv
	}

	defaultDir := filepath.Join(homeDir, ".golem", "builtin-skills")
	candidates := []string{
		defaultDir,
		filepath.Join(homeDir, ".golem", "golem", "skills"),
	}

	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "skills"))
	}

	// Source checkout fallback for local development.
	if _, thisFile, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "skills")))
	}

	for _, dir := range candidates {
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			return dir
		}
	}

	return defaultDir
}
