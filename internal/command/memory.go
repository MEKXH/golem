package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/memory"
)

// MemoryCommand implements /memory — read long-term memory and diary entries.
// Subcommands: (none)|read, diary [date|recent]
type MemoryCommand struct{}

func (c *MemoryCommand) Name() string        { return "memory" }
func (c *MemoryCommand) Description() string { return "Read memory (long-term or diary)" }

func (c *MemoryCommand) Execute(_ context.Context, args string, env Env) Result {
	sub, rest, _ := strings.Cut(args, " ")
	sub = strings.ToLower(strings.TrimSpace(sub))
	rest = strings.TrimSpace(rest)

	mgr := memory.NewManager(env.WorkspacePath)

	switch sub {
	case "", "read":
		return memoryRead(mgr)
	case "diary":
		return memoryDiary(mgr, rest)
	default:
		return Result{Content: "Usage: /memory [read|diary [YYYY-MM-DD|recent]]"}
	}
}

func memoryRead(mgr *memory.Manager) Result {
	content, err := mgr.ReadLongTerm()
	if err != nil {
		return Result{Content: fmt.Sprintf("Error: %v", err)}
	}
	if content == "" {
		return Result{Content: "Long-term memory is empty."}
	}
	if len(content) > 2000 {
		content = content[:2000] + "\n...(truncated)"
	}
	return Result{Content: content}
}

func memoryDiary(mgr *memory.Manager, dateOrRecent string) Result {
	dateOrRecent = strings.TrimSpace(dateOrRecent)

	// Default or "recent": show last 3 diary entries
	if dateOrRecent == "" || dateOrRecent == "recent" {
		entries, err := mgr.ReadRecentDiaries(3)
		if err != nil {
			return Result{Content: fmt.Sprintf("Error: %v", err)}
		}
		if len(entries) == 0 {
			return Result{Content: "No diary entries."}
		}
		var sb strings.Builder
		for _, e := range entries {
			sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", e.Date, e.Content))
		}
		return Result{Content: strings.TrimSpace(sb.String())}
	}

	// Specific date
	content, err := mgr.ReadDiary(dateOrRecent)
	if err != nil {
		return Result{Content: fmt.Sprintf("Error: %v", err)}
	}
	if content == "" {
		return Result{Content: fmt.Sprintf("No diary entry for %s.", dateOrRecent)}
	}
	return Result{Content: fmt.Sprintf("--- %s ---\n%s", dateOrRecent, content)}
}
