package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/MEKXH/golem/internal/memory"
	"github.com/MEKXH/golem/internal/metrics"
	"github.com/MEKXH/golem/internal/session"
	"github.com/MEKXH/golem/internal/skills"
	"github.com/cloudwego/eino/schema"
)

// ContextBuilder builds LLM context
type ContextBuilder struct {
	workspacePath   string
	runtimeMetrics  *metrics.RuntimeMetrics
	mu              sync.RWMutex
	cachedBaseParts []string
}

// NewContextBuilder creates a context builder
func NewContextBuilder(workspacePath string) *ContextBuilder {
	return &ContextBuilder{workspacePath: workspacePath}
}

// SetRuntimeMetrics attaches a runtime metrics recorder
func (c *ContextBuilder) SetRuntimeMetrics(recorder *metrics.RuntimeMetrics) {
	c.runtimeMetrics = recorder
}

// InvalidateCache clears the cached system prompt parts
func (c *ContextBuilder) InvalidateCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cachedBaseParts = nil
}

// BuildSystemPrompt assembles the system prompt
func (c *ContextBuilder) BuildSystemPrompt() string {
	parts := c.buildBaseSystemPromptParts()

	if mem := c.readWorkspaceFile(filepath.Join("memory", "MEMORY.md")); mem != "" {
		parts = append(parts, "## Long-term Memory\n"+mem)
	}

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
		// Return a copy to avoid modification by caller
		result := make([]string, len(c.cachedBaseParts))
		copy(result, c.cachedBaseParts)
		return result
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double check after acquiring write lock
	if c.cachedBaseParts != nil {
		result := make([]string, len(c.cachedBaseParts))
		copy(result, c.cachedBaseParts)
		return result
	}

	parts := make([]string, 0, 8)
	parts = append(parts, c.coreIdentity())

	bootstrapFiles := []string{"IDENTITY.md", "SOUL.md", "USER.md", "TOOLS.md", "AGENTS.md"}
	for _, name := range bootstrapFiles {
		if content := c.readWorkspaceFile(name); content != "" {
			parts = append(parts, "## "+strings.TrimSuffix(name, ".md")+"\n"+content)
		}
	}

	if skillsSummary := skills.NewLoader(c.workspacePath).BuildSkillsSummary(); skillsSummary != "" {
		parts = append(parts, skillsSummary)
	}

	// Store copy in cache
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

// BuildMessages constructs the full message list
func (c *ContextBuilder) BuildMessages(history []*session.Message, current string, media []string) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+2)
	currentContent := strings.TrimSpace(current)

	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: c.buildSystemPromptForInput(currentContent),
	})

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
