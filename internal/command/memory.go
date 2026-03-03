package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/memory"
)

// MemoryCommand 实现 /memory 命令 — 用于读取长期记忆 (MEMORY.md) 或查询日记条目。
// 使用方式:
//   /memory [read] - 读取长期记忆内容
//   /memory diary [date|recent] - 读取指定日期或最近的日记分录
type MemoryCommand struct{}

// Name 返回命令名称。
func (c *MemoryCommand) Name() string { return "memory" }

// Description 返回命令描述。
func (c *MemoryCommand) Description() string { return "Read memory (long-term or diary)" }

// Execute 执行记忆查询逻辑。
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
		return Result{Content: "Usage: `/memory [read|diary [YYYY-MM-DD|recent]]`"}
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
			sb.WriteString(fmt.Sprintf("**%s**\n\n%s\n\n", e.Date, e.Content))
		}
		return Result{Content: strings.TrimSpace(sb.String())}
	}

	content, err := mgr.ReadDiary(dateOrRecent)
	if err != nil {
		return Result{Content: fmt.Sprintf("Error: %v", err)}
	}
	if content == "" {
		return Result{Content: fmt.Sprintf("No diary entry for %s.", dateOrRecent)}
	}
	return Result{Content: fmt.Sprintf("**%s**\n\n%s", dateOrRecent, content)}
}
