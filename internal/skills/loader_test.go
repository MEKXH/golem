package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderListSkills_SourcePriority(t *testing.T) {
	tmp := t.TempDir()

	workspaceSkills := filepath.Join(tmp, "workspace", "skills")
	globalSkills := filepath.Join(tmp, "global", "skills")
	builtinSkills := filepath.Join(tmp, "builtin", "skills")

	mustWriteSkill(t, workspaceSkills, "same", "workspace wins")
	mustWriteSkill(t, globalSkills, "same", "global loses to workspace")
	mustWriteSkill(t, builtinSkills, "same", "builtin loses to workspace/global")

	mustWriteSkill(t, globalSkills, "global-only", "global skill")
	mustWriteSkill(t, builtinSkills, "builtin-only", "builtin skill")

	loader := &Loader{
		workspaceSkills: workspaceSkills,
		globalSkills:    globalSkills,
		builtinSkills:   builtinSkills,
	}

	skills := loader.ListSkills()
	byName := make(map[string]SkillInfo, len(skills))
	for _, s := range skills {
		byName[s.Name] = s
	}

	if got := byName["same"].Source; got != "workspace" {
		t.Fatalf("expected workspace to win for same, got source=%q", got)
	}
	if got := byName["global-only"].Source; got != "global" {
		t.Fatalf("expected global-only from global, got source=%q", got)
	}
	if got := byName["builtin-only"].Source; got != "builtin" {
		t.Fatalf("expected builtin-only from builtin, got source=%q", got)
	}
}

func TestLoaderLoadSkill_FallsBackToBuiltin(t *testing.T) {
	tmp := t.TempDir()
	builtinSkills := filepath.Join(tmp, "builtin", "skills")
	mustWriteSkill(t, builtinSkills, "weather", "builtin weather")

	loader := &Loader{
		workspaceSkills: filepath.Join(tmp, "workspace", "skills"),
		globalSkills:    filepath.Join(tmp, "global", "skills"),
		builtinSkills:   builtinSkills,
	}

	content, err := loader.LoadSkill("weather")
	if err != nil {
		t.Fatalf("LoadSkill returned error: %v", err)
	}
	if !strings.Contains(content, "builtin weather") {
		t.Fatalf("expected builtin skill content, got: %s", content)
	}
}

func mustWriteSkill(t *testing.T, baseDir, name, description string) {
	t.Helper()
	dir := filepath.Join(baseDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	content := "---\nname: " + name + "\ndescription: \"" + description + "\"\n---\n\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%s): %v", name, err)
	}
}
