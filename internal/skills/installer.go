package skills

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Installer manages skill installation and removal.
type Installer struct {
	skillsDir string // workspace/skills/
}

// NewInstaller creates an installer targeting the given workspace.
func NewInstaller(workspacePath string) *Installer {
	return &Installer{
		skillsDir: filepath.Join(workspacePath, "skills"),
	}
}

// Install downloads a SKILL.md from a GitHub repository.
// repo format: "owner/repo" or "owner/repo/path/to/SKILL.md"
// The skill name is derived from the repo name by default.
func (i *Installer) Install(ctx context.Context, repo string) error {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return fmt.Errorf("repo is required")
	}

	// Remove github.com prefix if present.
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "github.com/")
	repo = strings.TrimSuffix(repo, "/")

	parts := strings.SplitN(repo, "/", 3)
	if len(parts) < 2 {
		return fmt.Errorf("invalid repo format, expected owner/repo: %s", repo)
	}

	owner := parts[0]
	repoName := parts[1]
	filePath := "SKILL.md"
	if len(parts) == 3 && parts[2] != "" {
		filePath = parts[2]
	}

	// Derive skill name from repo or directory name.
	skillName := repoName
	if len(parts) == 3 {
		dir := filepath.Dir(parts[2])
		if dir != "." && dir != "" {
			skillName = filepath.Base(dir)
		}
	}

	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", owner, repoName, filePath)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch skill failed: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return fmt.Errorf("read skill content: %w", err)
	}

	destDir := filepath.Join(i.skillsDir, skillName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}

	destPath := filepath.Join(destDir, "SKILL.md")
	if err := os.WriteFile(destPath, body, 0644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	return nil
}

// Uninstall removes a skill by name.
func (i *Installer) Uninstall(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("skill name is required")
	}

	skillDir := filepath.Join(i.skillsDir, name)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill not found: %s", name)
	}

	return os.RemoveAll(skillDir)
}
