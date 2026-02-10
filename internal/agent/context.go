package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MEKXH/golem/internal/memory"
	"github.com/MEKXH/golem/internal/session"
	"github.com/cloudwego/eino/schema"
)

// ContextBuilder builds LLM context
type ContextBuilder struct {
	workspacePath string
}

// NewContextBuilder creates a context builder
func NewContextBuilder(workspacePath string) *ContextBuilder {
	return &ContextBuilder{workspacePath: workspacePath}
}

// BuildSystemPrompt assembles the system prompt
func (c *ContextBuilder) BuildSystemPrompt() string {
	var parts []string

	parts = append(parts, c.coreIdentity())

	bootstrapFiles := []string{"IDENTITY.md", "SOUL.md", "USER.md", "TOOLS.md", "AGENTS.md"}
	for _, name := range bootstrapFiles {
		if content := c.readWorkspaceFile(name); content != "" {
			parts = append(parts, "## "+strings.TrimSuffix(name, ".md")+"\n"+content)
		}
	}

	if mem := c.readWorkspaceFile(filepath.Join("memory", "MEMORY.md")); mem != "" {
		parts = append(parts, "## Long-term Memory\n"+mem)
	}

	if diary := c.buildRecentDiarySection(); diary != "" {
		parts = append(parts, diary)
	}

	return strings.Join(parts, "\n\n")
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

// BuildMessages constructs the full message list
func (c *ContextBuilder) BuildMessages(history []*session.Message, current string, media []string) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+2)

	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: c.BuildSystemPrompt(),
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

	content := strings.TrimSpace(current)
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
