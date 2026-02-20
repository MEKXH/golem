package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultSkillsIndexURL = "https://raw.githubusercontent.com/MEKXH/golem-skills/main/skills.json"
const defaultGitHubRawBaseURL = "https://raw.githubusercontent.com"

// AvailableSkill is an entry from the remote skills index.
type AvailableSkill struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

// Installer manages skill installation and removal.
type Installer struct {
	skillsDir      string // workspace/skills/
	httpClient     *http.Client
	skillsIndexURL string
	githubRawBase  string
}

// NewInstaller creates an installer targeting the given workspace.
func NewInstaller(workspacePath string) *Installer {
	return &Installer{
		skillsDir: filepath.Join(workspacePath, "skills"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		skillsIndexURL: resolveSkillsIndexURL(),
		githubRawBase:  resolveGitHubRawBaseURL(),
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
	repo = strings.TrimPrefix(repo, "http://github.com/")
	repo = strings.TrimPrefix(repo, "github.com/")
	repo = strings.TrimSuffix(repo, "/")

	parts := strings.SplitN(repo, "/", 3)
	if len(parts) < 2 {
		return fmt.Errorf("invalid repo format, expected owner/repo: %s", repo)
	}

	owner := parts[0]
	repoName := parts[1]
	filePath := ""
	if len(parts) == 3 && parts[2] != "" {
		filePath = parts[2]
	}

	// Strip GitHub web UI path segments: tree/<branch>/ or blob/<branch>/
	filePath = stripGitHubBranchPrefix(filePath)

	// If filePath doesn't point to a .md file, treat it as a directory.
	if !strings.HasSuffix(strings.ToLower(filePath), ".md") {
		filePath = strings.TrimSuffix(filePath, "/")
		if filePath == "" {
			filePath = "SKILL.md"
		} else {
			filePath = filePath + "/SKILL.md"
		}
	}

	// Derive skill name from repo or directory name.
	skillName := repoName
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		skillName = filepath.Base(dir)
	}

	rawURL := fmt.Sprintf("%s/%s/%s/main/%s", strings.TrimRight(i.githubRawBase, "/"), owner, repoName, filePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := i.httpClient.Do(req)
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

// Search returns available skills from the configured index URL.
func (i *Installer) Search(ctx context.Context) ([]AvailableSkill, error) {
	if strings.TrimSpace(i.skillsIndexURL) == "" {
		return nil, fmt.Errorf("skills index URL is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, i.skillsIndexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch skills index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch skills index failed: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read skills index: %w", err)
	}

	var list []AvailableSkill
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("parse skills index: %w", err)
	}
	return list, nil
}

// stripGitHubBranchPrefix removes "tree/<branch>/" or "blob/<branch>/" from
// a path that was copied from a GitHub web URL.
// e.g. "tree/main/skills/find-skills" → "skills/find-skills"
func stripGitHubBranchPrefix(path string) string {
	for _, prefix := range []string{"tree/", "blob/"} {
		if strings.HasPrefix(path, prefix) {
			// Remove "tree/" or "blob/", then strip the branch name segment.
			rest := strings.TrimPrefix(path, prefix)
			if idx := strings.Index(rest, "/"); idx >= 0 {
				return rest[idx+1:]
			}
			// Only "tree/<branch>" with no trailing path — nothing left.
			return ""
		}
	}
	return path
}

func resolveSkillsIndexURL() string {
	if fromEnv := strings.TrimSpace(os.Getenv("GOLEM_SKILLS_INDEX_URL")); fromEnv != "" {
		return fromEnv
	}
	return defaultSkillsIndexURL
}

func resolveGitHubRawBaseURL() string {
	if fromEnv := strings.TrimSpace(os.Getenv("GOLEM_GITHUB_RAW_BASE_URL")); fromEnv != "" {
		return fromEnv
	}
	return defaultGitHubRawBaseURL
}
