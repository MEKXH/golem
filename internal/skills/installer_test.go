package skills

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInstallerSearch_ReturnsAvailableSkills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"weather","repository":"owner/weather","description":"Weather lookup","author":"dev","tags":["tooling"]},
			{"name":"summarize","repository":"owner/summarize","description":"Summaries","author":"dev2","tags":["text"]}
		]`))
	}))
	defer srv.Close()

	installer := &Installer{
		httpClient:     srv.Client(),
		skillsIndexURL: srv.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	items, err := installer.Search(ctx)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(items))
	}
	if items[0].Name != "weather" || items[0].Repository != "owner/weather" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
}

func TestStripGitHubBranchPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"skills/weather", "skills/weather"},
		{"tree/main/skills/find-skills", "skills/find-skills"},
		{"tree/master/src/SKILL.md", "src/SKILL.md"},
		{"tree/dev", ""},
		{"blob/main/path/to/SKILL.md", "path/to/SKILL.md"},
		{"tree/v2.0/deep/nested/path", "deep/nested/path"},
		{"treefoo/bar", "treefoo/bar"},
	}
	for _, tt := range tests {
		got := stripGitHubBranchPrefix(tt.input)
		if got != tt.want {
			t.Errorf("stripGitHubBranchPrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInstall_URLConstruction(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPath   string // expected request path on the mock server
		wantSkill  string // expected skill directory name
	}{
		{
			name:      "simple owner/repo",
			input:     "owner/repo",
			wantPath:  "/owner/repo/main/SKILL.md",
			wantSkill: "repo",
		},
		{
			name:      "owner/repo with subdirectory",
			input:     "owner/repo/skills/weather",
			wantPath:  "/owner/repo/main/skills/weather/SKILL.md",
			wantSkill: "weather",
		},
		{
			name:      "full GitHub URL with tree/main",
			input:     "https://github.com/vercel-labs/skills/tree/main/skills/find-skills",
			wantPath:  "/vercel-labs/skills/main/skills/find-skills/SKILL.md",
			wantSkill: "find-skills",
		},
		{
			name:      "GitHub URL with blob/main",
			input:     "https://github.com/owner/repo/blob/main/custom/SKILL.md",
			wantPath:  "/owner/repo/main/custom/SKILL.md",
			wantSkill: "custom",
		},
		{
			name:      "path with tree/master branch",
			input:     "owner/repo/tree/master/plugins/my-skill",
			wantPath:  "/owner/repo/main/plugins/my-skill/SKILL.md",
			wantSkill: "my-skill",
		},
		{
			name:      "explicit SKILL.md in path",
			input:     "owner/repo/skills/weather/SKILL.md",
			wantPath:  "/owner/repo/main/skills/weather/SKILL.md",
			wantSkill: "weather",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("# Test Skill\n"))
			}))
			defer srv.Close()

			tmpDir := t.TempDir()
			installer := &Installer{
				skillsDir:     filepath.Join(tmpDir, "skills"),
				httpClient:    srv.Client(),
				githubRawBase: srv.URL,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := installer.Install(ctx, tt.input)
			if err != nil {
				t.Fatalf("Install(%q) returned error: %v", tt.input, err)
			}

			if gotPath != tt.wantPath {
				t.Errorf("request path = %q, want %q", gotPath, tt.wantPath)
			}

			// Verify skill file was written to expected directory.
			skillFile := filepath.Join(tmpDir, "skills", tt.wantSkill, "SKILL.md")
			if _, err := os.Stat(skillFile); os.IsNotExist(err) {
				t.Errorf("expected skill file at %s, but it does not exist", skillFile)
			}
		})
	}
}

func TestInstallerSearch_Non200ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	installer := &Installer{
		httpClient:     srv.Client(),
		skillsIndexURL: srv.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := installer.Search(ctx); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}
