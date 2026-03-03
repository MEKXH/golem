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

// AvailableSkill 表示远程技能索引中的一个可用技能条目。
type AvailableSkill struct {
	Name        string   `json:"name"`        // 技能名称
	Repository  string   `json:"repository"`  // 仓库地址 (owner/repo)
	Description string   `json:"description"` // 技能描述
	Author      string   `json:"author"`      // 作者
	Tags        []string `json:"tags"`        // 标签列表
}

// Installer 负责从远程仓库安装或卸载技能。
type Installer struct {
	skillsDir      string       // 本地技能存储目录
	httpClient     *http.Client // 用于下载的 HTTP 客户端
	skillsIndexURL string       // 技能索引 JSON 的 URL
	githubRawBase  string       // GitHub Raw 内容的基准 URL
}

// NewInstaller 为指定的工作区创建一个新的技能安装器。
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

// Install 从指定的 GitHub 仓库下载并安装技能 (SKILL.md)。
// 仓库参数格式支持: "owner/repo" 或完整的 GitHub URL。
func (i *Installer) Install(ctx context.Context, repo string) error {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return fmt.Errorf("repo is required")
	}

	// 预处理仓库地址，移除前缀
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

	// 移除 GitHub 网页链接中的 tree/<branch>/ 或 blob/<branch>/ 路径段
	filePath = stripGitHubBranchPrefix(filePath)

	// 如果未指定具体文件，则默认寻找目录下的 SKILL.md
	if !strings.HasSuffix(strings.ToLower(filePath), ".md") {
		filePath = strings.TrimSuffix(filePath, "/")
		if filePath == "" {
			filePath = "SKILL.md"
		} else {
			filePath = filePath + "/SKILL.md"
		}
	}

	// 从路径或仓库名推导技能名称
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 限制最大 1MB
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

// Uninstall 根据技能名称从本地工作区移除已安装的技能。
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

// Search 从配置的索引地址获取所有可用的远程技能列表。
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

// stripGitHubBranchPrefix 移除从 GitHub 网页 URL 中携带的目录前缀（如 "tree/main/"）。
func stripGitHubBranchPrefix(path string) string {
	for _, prefix := range []string{"tree/", "blob/"} {
		if strings.HasPrefix(path, prefix) {
			rest := strings.TrimPrefix(path, prefix)
			if idx := strings.Index(rest, "/"); idx >= 0 {
				return rest[idx+1:]
			}
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
