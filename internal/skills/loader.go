// Package skills 实现 Golem 的增强技能系统，允许通过外部 Markdown 文件定义复杂指令与工作流。
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SkillInfo 描述已加载技能的元数据。
type SkillInfo struct {
	Name        string `json:"name"`        // 技能名称
	Description string `json:"description"` // 技能功能描述
	Path        string `json:"path"`        // 技能文件 (SKILL.md) 的绝对路径
	Source      string `json:"source"`      // 来源："workspace" | "global" | "builtin"
}

// Loader 负责发现、扫描并加载不同来源的技能文件。
type Loader struct {
	workspaceSkills string // 工作区技能目录: <workspace>/skills/
	globalSkills    string // 全局用户技能目录: ~/.golem/skills/
	builtinSkills   string // 随 Golem 发行的内置技能目录
}

// NewLoader 为指定的工作区路径创建一个新的技能加载器。
func NewLoader(workspacePath string) *Loader {
	homeDir, _ := os.UserHomeDir()
	return &Loader{
		workspaceSkills: filepath.Join(workspacePath, "skills"),
		globalSkills:    filepath.Join(homeDir, ".golem", "skills"),
		builtinSkills:   resolveBuiltinSkillsDir(homeDir),
	}
}

// ListSkills 扫描并返回所有已发现的技能，按优先级排序（工作区 > 全局 > 内置）。
func (l *Loader) ListSkills() []SkillInfo {
	seen := make(map[string]bool)
	var skills []SkillInfo

	// 工作区技能 (最高优先级)
	for _, s := range l.scanDir(l.workspaceSkills, "workspace") {
		seen[s.Name] = true
		skills = append(skills, s)
	}

	// 全局用户技能
	for _, s := range l.scanDir(l.globalSkills, "global") {
		if !seen[s.Name] {
			seen[s.Name] = true
			skills = append(skills, s)
		}
	}

	// 内置技能
	for _, s := range l.scanDir(l.builtinSkills, "builtin") {
		if !seen[s.Name] {
			skills = append(skills, s)
		}
	}

	return skills
}

// LoadSkill 根据技能名称读取并返回其 Markdown 内容。
// 它会按优先级顺序在各个技能目录中搜索。
func (l *Loader) LoadSkill(name string) (string, error) {
	// 依次搜索工作区、全局和内置目录
	for _, dir := range []string{l.workspaceSkills, l.globalSkills, l.builtinSkills} {
		path := filepath.Join(dir, name, "SKILL.md")
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("skill not found: %s", name)
}

// BuildSkillsSummary 生成所有已安装技能的格式化摘要字符串，通常用于注入到系统提示词 (System Prompt) 中。
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

// parseSkillFrontmatter 从技能 Markdown 文件的 YAML frontmatter 中提取名称和描述。
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

	// 源码环境回退（用于开发环境）
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
