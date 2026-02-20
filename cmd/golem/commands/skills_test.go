package commands

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/config"
)

func TestSkillsList_IncludesBuiltinSource(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		t.Fatalf("WorkspacePathChecked: %v", err)
	}

	builtinDir := filepath.Join(tmpDir, "builtin-skills")
	t.Setenv("GOLEM_BUILTIN_SKILLS_DIR", builtinDir)
	if err := os.MkdirAll(filepath.Join(builtinDir, "weather"), 0755); err != nil {
		t.Fatalf("MkdirAll builtin weather: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(builtinDir, "weather", "SKILL.md"),
		[]byte("---\nname: weather\ndescription: \"builtin weather\"\n---\n"),
		0644,
	); err != nil {
		t.Fatalf("WriteFile builtin SKILL.md: %v", err)
	}

	_ = workspacePath // explicit: workspace exists, loader will read from config workspace
	out := captureOutput(t, func() {
		if err := runSkillsList(nil, nil); err != nil {
			t.Fatalf("runSkillsList: %v", err)
		}
	})
	if !strings.Contains(out, "builtin") {
		t.Fatalf("expected builtin source in output, got: %s", out)
	}
	if !strings.Contains(out, "weather") {
		t.Fatalf("expected builtin skill name in output, got: %s", out)
	}
}

func TestSkillsSearch_PrintsResults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"weather","repository":"owner/weather","description":"Weather lookup","author":"dev","tags":["forecast"]}
		]`))
	}))
	defer srv.Close()

	t.Setenv("GOLEM_SKILLS_INDEX_URL", srv.URL)

	out := captureOutput(t, func() {
		if err := runSkillsSearch(nil, nil); err != nil {
			t.Fatalf("runSkillsSearch: %v", err)
		}
	})
	if !strings.Contains(out, "weather") {
		t.Fatalf("expected weather in output, got: %s", out)
	}
	if !strings.Contains(out, "owner/weather") {
		t.Fatalf("expected repository in output, got: %s", out)
	}
}

func TestSkillsCommands_EndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/acme/demo-skill/main/SKILL.md":
			w.Header().Set("Content-Type", "text/markdown")
			_, _ = w.Write([]byte(`---
name: demo-skill
description: "Demo skill from test server"
---

# Demo Skill
`))
		case "/skills.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"name":"demo-skill","repository":"acme/demo-skill","description":"Demo skill from test server","author":"acme","tags":["demo"]}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Setenv("GOLEM_GITHUB_RAW_BASE_URL", srv.URL)
	t.Setenv("GOLEM_SKILLS_INDEX_URL", srv.URL+"/skills.json")

	outInstall := captureOutput(t, func() {
		if err := runSkillsInstall(nil, []string{"acme/demo-skill"}); err != nil {
			t.Fatalf("runSkillsInstall: %v", err)
		}
	})
	if !strings.Contains(outInstall, "Skill installed successfully.") {
		t.Fatalf("expected install success output, got: %s", outInstall)
	}

	outList := captureOutput(t, func() {
		if err := runSkillsList(nil, nil); err != nil {
			t.Fatalf("runSkillsList: %v", err)
		}
	})
	if !strings.Contains(outList, "demo-skill") {
		t.Fatalf("expected installed skill in list, got: %s", outList)
	}

	outShow := captureOutput(t, func() {
		if err := runSkillsShow(nil, []string{"demo-skill"}); err != nil {
			t.Fatalf("runSkillsShow: %v", err)
		}
	})
	if !strings.Contains(outShow, "Demo Skill") {
		t.Fatalf("expected skill content in show output, got: %s", outShow)
	}

	outSearch := captureOutput(t, func() {
		if err := runSkillsSearch(nil, []string{"demo"}); err != nil {
			t.Fatalf("runSkillsSearch: %v", err)
		}
	})
	if !strings.Contains(outSearch, "acme/demo-skill") {
		t.Fatalf("expected search result in output, got: %s", outSearch)
	}

	outRemove := captureOutput(t, func() {
		if err := runSkillsRemove(nil, []string{"demo-skill"}); err != nil {
			t.Fatalf("runSkillsRemove: %v", err)
		}
	})
	if !strings.Contains(outRemove, "removed") {
		t.Fatalf("expected remove output, got: %s", outRemove)
	}

	outListAfterRemove := captureOutput(t, func() {
		if err := runSkillsList(nil, nil); err != nil {
			t.Fatalf("runSkillsList after remove: %v", err)
		}
	})
	if strings.Contains(outListAfterRemove, "demo-skill") {
		t.Fatalf("did not expect removed skill in list output, got: %s", outListAfterRemove)
	}
}
