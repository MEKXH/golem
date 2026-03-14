package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/MEKXH/golem/internal/geocodebook"
	"github.com/MEKXH/golem/internal/geopipeline"
	"github.com/MEKXH/golem/internal/geotoolfab"
	"github.com/MEKXH/golem/internal/memory"
	"github.com/MEKXH/golem/internal/metrics"
	"github.com/MEKXH/golem/internal/session"
	"github.com/MEKXH/golem/internal/skills"
	"github.com/cloudwego/eino/schema"
)

// watchedBaseFiles 定义了 ContextBuilder 监控的工作区基础 Markdown 文件列表。
var watchedBaseFiles = []string{
	"IDENTITY.md", "SOUL.md", "USER.md", "TOOLS.md", "AGENTS.md",
}

// ContextBuilder 负责根据配置、历史记录和外部文件构建 LLM 的 Prompt 上下文。
type ContextBuilder struct {
	workspacePath   string                  // 工作区根路径
	runtimeMetrics  *metrics.RuntimeMetrics // 运行时指标记录器
	mu              sync.RWMutex
	cachedBaseParts []string // 缓存的基础 Prompt 片段
}

// NewContextBuilder 创建并返回一个新的上下文构建器。
func NewContextBuilder(workspacePath string) *ContextBuilder {
	return &ContextBuilder{workspacePath: workspacePath}
}

// SetRuntimeMetrics 为构建器关联一个运行时指标记录器。
func (c *ContextBuilder) SetRuntimeMetrics(recorder *metrics.RuntimeMetrics) {
	c.runtimeMetrics = recorder
}

// InvalidateCache 根据发生变化的文件路径使缓存失效。
// 如果 changedPath 为空，则强制使所有基础缓存失效。
func (c *ContextBuilder) InvalidateCache(changedPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if changedPath == "" {
		c.cachedBaseParts = nil
		return
	}

	workspaceAbs, err := filepath.Abs(c.workspacePath)
	if err != nil {
		c.cachedBaseParts = nil
		return
	}

	changedAbs, err := filepath.Abs(changedPath)
	if err != nil {
		c.cachedBaseParts = nil
		return
	}

	// 检查是否属于基础监控文件
	for _, name := range watchedBaseFiles {
		if changedAbs == filepath.Join(workspaceAbs, name) {
			c.cachedBaseParts = nil
			return
		}
	}

	// 检查是否属于技能目录下的文件
	skillsDir := filepath.Join(workspaceAbs, "skills")
	rel, err := filepath.Rel(skillsDir, changedAbs)
	if err == nil && !strings.HasPrefix(rel, "..") {
		c.cachedBaseParts = nil
		return
	}

	codebookDir := filepath.Join(workspaceAbs, "geo-codebook")
	rel, err = filepath.Rel(codebookDir, changedAbs)
	if err == nil && !strings.HasPrefix(rel, "..") {
		c.cachedBaseParts = nil
		return
	}

	fabricatedToolsDir := filepath.Join(workspaceAbs, "tools", "geo")
	rel, err = filepath.Rel(fabricatedToolsDir, changedAbs)
	if err == nil && !strings.HasPrefix(rel, "..") {
		c.cachedBaseParts = nil
		return
	}

	learnedPipelinesDir := filepath.Join(workspaceAbs, "pipelines", "geo")
	rel, err = filepath.Rel(learnedPipelinesDir, changedAbs)
	if err == nil && !strings.HasPrefix(rel, "..") {
		c.cachedBaseParts = nil
		return
	}
}

// BuildSystemPrompt 组装完整的系统提示词 (System Prompt)。
func (c *ContextBuilder) BuildSystemPrompt() string {
	parts := c.buildBaseSystemPromptParts()

	// 注入长期记忆
	if mem := c.readWorkspaceFile(filepath.Join("memory", "MEMORY.md")); mem != "" {
		parts = append(parts, "## Long-term Memory\n"+mem)
	}

	// 注入最近的日记
	if diary := c.buildRecentDiarySection(); diary != "" {
		parts = append(parts, diary)
	}

	return strings.Join(parts, "\n\n")
}

func (c *ContextBuilder) buildSystemPromptForInput(query string) string {
	parts := c.buildBaseSystemPromptParts()
	if recall := c.buildMemoryRecallSection(query); recall != "" {
		parts = append(parts, recall)
	}
	return strings.Join(parts, "\n\n")
}

func (c *ContextBuilder) buildBaseSystemPromptParts() []string {
	c.mu.RLock()
	if c.cachedBaseParts != nil {
		defer c.mu.RUnlock()
		result := make([]string, len(c.cachedBaseParts))
		copy(result, c.cachedBaseParts)
		return result
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedBaseParts != nil {
		result := make([]string, len(c.cachedBaseParts))
		copy(result, c.cachedBaseParts)
		return result
	}

	parts := make([]string, 0, 8)
	parts = append(parts, c.coreIdentity())

	// 加载并添加所有监控的基础 Markdown 文件内容
	for _, name := range watchedBaseFiles {
		if content := c.readWorkspaceFile(name); content != "" {
			parts = append(parts, "## "+strings.TrimSuffix(name, ".md")+"\n"+content)
		}
	}

	// 注入技能摘要
	if skillsSummary := skills.NewLoader(c.workspacePath).BuildSkillsSummary(); skillsSummary != "" {
		parts = append(parts, skillsSummary)
	}

	if codebookSummary, err := geocodebook.NewLoader(c.workspacePath).BuildSummary(); err == nil && codebookSummary != "" {
		parts = append(parts, codebookSummary)
	}

	if fabricatedSummary := geotoolfab.NewLoader(c.workspacePath).BuildSummary(); fabricatedSummary != "" {
		parts = append(parts, fabricatedSummary)
	}

	if pipelineSummary := geopipeline.NewRecorder(c.workspacePath).BuildSummary(); pipelineSummary != "" {
		parts = append(parts, pipelineSummary)
	}

	c.cachedBaseParts = make([]string, len(parts))
	copy(c.cachedBaseParts, parts)

	return parts
}

func (c *ContextBuilder) coreIdentity() string {
	return `You are Golem, a personal AI assistant.
You have access to tools for file operations, shell commands, and more.
Be helpful, concise, and proactive. Use tools when needed to accomplish tasks.`
}

func (c *ContextBuilder) readWorkspaceFile(name string) string {
	path := filepath.Join(c.workspacePath, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (c *ContextBuilder) buildRecentDiarySection() string {
	memMgr := memory.NewManager(c.workspacePath)
	entries, err := memMgr.ReadRecentDiaries(3)
	if err != nil || len(entries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Recent Diary Entries")
	for _, entry := range entries {
		sb.WriteString(fmt.Sprintf("\n\n### %s\n%s", entry.Date, entry.Content))
	}
	return sb.String()
}

func (c *ContextBuilder) buildMemoryRecallSection(query string) string {
	memMgr := memory.NewManager(c.workspacePath)
	recall, err := memMgr.RecallContext(query, 3, 3)
	if err != nil || recall.RecallCount == 0 {
		return ""
	}

	if c.runtimeMetrics != nil {
		_, _ = c.runtimeMetrics.RecordMemoryRecall(recall.RecallCount, recall.SourceHits)
	}

	var sb strings.Builder
	sb.WriteString("## Memory Recall")
	sb.WriteString(fmt.Sprintf("\nquery: %s", strings.TrimSpace(query)))
	sb.WriteString(fmt.Sprintf("\nrecall_count: %d", recall.RecallCount))
	sb.WriteString(fmt.Sprintf("\nhit_sources: %s", formatSourceHits(recall.SourceHits)))

	for i, item := range recall.Items {
		label := item.Source
		if item.Date != "" {
			label = label + ":" + item.Date
		}
		sb.WriteString(fmt.Sprintf("\n\n[%d] %s\n%s", i+1, label, strings.TrimSpace(item.Excerpt)))
	}
	return sb.String()
}

func formatSourceHits(sourceHits map[string]int) string {
	if len(sourceHits) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(sourceHits))
	for key, count := range sourceHits {
		if count > 0 {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return "none"
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, sourceHits[key]))
	}
	return strings.Join(parts, ", ")
}

// BuildMessages 根据历史记录、当前输入及媒体附件构建发送给 LLM 的完整消息列表。
func (c *ContextBuilder) BuildMessages(history []*session.Message, current string, media []string) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+2)
	currentContent := strings.TrimSpace(current)

	// 注入动态构建的系统提示词
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: c.buildSystemPromptForInput(currentContent),
	})

	// 注入会话历史
	for _, h := range history {
		role := schema.User
		if h.Role == "assistant" {
			role = schema.Assistant
		}
		messages = append(messages, &schema.Message{
			Role:    role,
			Content: h.Content,
		})
	}

	// 注入当前用户输入及媒体信息
	content := currentContent
	if len(media) > 0 {
		var mb strings.Builder
		for _, item := range media {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			mb.WriteString("- " + item + "\n")
		}
		if mb.Len() > 0 {
			if content != "" {
				content += "\n\n"
			}
			content += "Attached media:\n" + strings.TrimRight(mb.String(), "\n")
		}
	}

	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: content,
	})

	return messages
}
