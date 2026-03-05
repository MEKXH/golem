// Package memory 实现 Golem 的记忆管理系统，包括长期记忆 (MEMORY.md) 和基于日记的短期记忆。
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	memoryDirName  = "memory"    // 记忆文件存储目录名
	memoryFileName = "MEMORY.md" // 长期记忆文件名
)

// DiaryEntry 表示单条日记分录。
type DiaryEntry struct {
	Date    string // 日期 (YYYY-MM-DD)
	Path    string // 文件路径
	Content string // 分录内容
}

// RecallItem 表示带来源归属的单条回忆片段。
type RecallItem struct {
	Source  string // 来源（如 "diary_recent", "long_term"）
	Date    string // 关联日期（可选）
	Path    string // 来源文件路径
	Excerpt string // 摘录内容
}

// RecallResult 总结了上下文回忆的质量和来源分布。
type RecallResult struct {
	Query       string         // 原始查询
	RecallCount int            // 召回的总片段数
	SourceHits  map[string]int // 各来源的命中统计
	Items       []RecallItem   // 召回的具体片段列表
}

type diaryFile struct {
	date string
	path string
}

// Manager 负责管理工作区内的记忆文件读写与上下文召回。
type Manager struct {
	workspacePath string
	memoryDir     string
	memoryFile    string
}

// NewManager 为指定的工作区创建一个记忆管理器。
func NewManager(workspacePath string) *Manager {
	memoryDir := filepath.Join(workspacePath, memoryDirName)
	return &Manager{
		workspacePath: workspacePath,
		memoryDir:     memoryDir,
		memoryFile:    filepath.Join(memoryDir, memoryFileName),
	}
}

// Ensure 确保记忆目录和长期记忆文件存在。
func (m *Manager) Ensure() error {
	if err := os.MkdirAll(m.memoryDir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(m.memoryFile); os.IsNotExist(err) {
		if err := os.WriteFile(m.memoryFile, []byte(""), 0644); err != nil {
			return err
		}
	}
	return nil
}

// ReadLongTerm 读取长期记忆文件的全文内容。
func (m *Manager) ReadLongTerm() (string, error) {
	data, err := os.ReadFile(m.memoryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteLongTerm 覆盖写入长期记忆文件。
func (m *Manager) WriteLongTerm(content string) error {
	if err := m.Ensure(); err != nil {
		return err
	}
	return os.WriteFile(m.memoryFile, []byte(strings.TrimSpace(content)), 0644)
}

// AppendDiary 在当前日期的日记文件中追加一条记录。
func (m *Manager) AppendDiary(entry string) (string, error) {
	return m.AppendDiaryAt(time.Now(), entry)
}

// AppendDiaryAt 在指定时间的日记文件中追加一条记录。
func (m *Manager) AppendDiaryAt(ts time.Time, entry string) (string, error) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return "", fmt.Errorf("entry is required")
	}
	if err := m.Ensure(); err != nil {
		return "", err
	}

	date := ts.Format("2006-01-02")
	path := filepath.Join(m.memoryDir, date+".md")
	line := fmt.Sprintf("- [%s] %s\n", ts.Format("15:04:05"), entry)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return "", err
	}
	return path, nil
}

// ReadDiary 读取指定日期的日记内容。
func (m *Manager) ReadDiary(date string) (string, error) {
	date = strings.TrimSpace(date)
	if date == "" {
		return "", fmt.Errorf("date is required")
	}
	path := filepath.Join(m.memoryDir, date+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ReadRecentDiaries 读取最近几天的日记分录。
func (m *Manager) ReadRecentDiaries(limit int) ([]DiaryEntry, error) {
	if limit <= 0 {
		limit = 3
	}
	diaries, err := m.collectDiaryFiles()
	if err != nil {
		return nil, err
	}

	// Optimization: Only do double sorting if we actually need to truncate.
	// Otherwise, a single ascending sort is sufficient.
	if len(diaries) > limit {
		sort.Slice(diaries, func(i, j int) bool {
			return diaries[i].date > diaries[j].date
		})
		diaries = diaries[:limit]
		sort.Slice(diaries, func(i, j int) bool {
			return diaries[i].date < diaries[j].date
		})
	} else {
		sort.Slice(diaries, func(i, j int) bool {
			return diaries[i].date < diaries[j].date
		})
	}

	out := make([]DiaryEntry, 0, len(diaries))
	for _, d := range diaries {
		data, err := os.ReadFile(d.path)
		if err != nil {
			return nil, err
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		out = append(out, DiaryEntry{
			Date:    d.date,
			Path:    d.path,
			Content: content,
		})
	}
	return out, nil
}

// RecallContext 使用“最近优先 + 关键词命中”策略从记忆中检索相关的上下文片段。
func (m *Manager) RecallContext(query string, recentLimit, keywordLimit int) (RecallResult, error) {
	if recentLimit <= 0 {
		recentLimit = 3
	}
	if keywordLimit <= 0 {
		keywordLimit = 3
	}

	result := RecallResult{
		Query:      strings.TrimSpace(query),
		SourceHits: map[string]int{},
		Items:      make([]RecallItem, 0, recentLimit+keywordLimit+1),
	}
	seenPaths := map[string]bool{}

	// 1. 预先收集所有日记文件
	diaries, err := m.collectDiaryFiles()
	if err != nil {
		return RecallResult{}, err
	}
	sort.Slice(diaries, func(i, j int) bool {
		return diaries[i].date > diaries[j].date // 最新的在前
	})

	// 2. 优先提取最近的日记分录
	recentCount := 0
	for _, d := range diaries {
		if recentCount >= recentLimit {
			break
		}

		contentRaw, readErr := os.ReadFile(d.path)
		if readErr != nil {
			continue
		}
		content := strings.TrimSpace(string(contentRaw))
		if content == "" {
			continue
		}

		result.SourceHits["diary_recent"]++
		seenPaths[d.path] = true
		result.Items = append(result.Items, RecallItem{
			Source:  "diary_recent",
			Date:    d.date,
			Path:    d.path,
			Excerpt: clipText(content, 320),
		})
		recentCount++
	}

	keywords := extractRecallKeywords(result.Query)
	if len(keywords) == 0 {
		result.RecallCount = len(result.Items)
		return result, nil
	}

	// 3. 检索长期记忆
	longTerm, err := m.ReadLongTerm()
	if err != nil {
		return RecallResult{}, err
	}
	if longTerm != "" {
		longTermLower := strings.ToLower(longTerm)
		if containsAnyKeyword(longTermLower, keywords) {
			result.SourceHits["long_term"]++
			result.Items = append(result.Items, RecallItem{
				Source:  "long_term",
				Date:    "",
				Path:    m.memoryFile,
				Excerpt: extractKeywordExcerpt(longTerm, longTermLower, keywords, 380),
			})
		}
	}

	// 4. 对剩余日记进行关键词搜索
	addedKeywordItems := 0
	for _, d := range diaries {
		if addedKeywordItems >= keywordLimit {
			break
		}

		// 优化：跳过已作为“最近”处理过的文件
		if seenPaths[d.path] {
			continue
		}

		contentRaw, readErr := os.ReadFile(d.path)
		if readErr != nil {
			continue
		}
		content := strings.TrimSpace(string(contentRaw))
		if content == "" {
			continue
		}
		contentLower := strings.ToLower(content)
		if !containsAnyKeyword(contentLower, keywords) {
			continue
		}

		result.SourceHits["diary_keyword"]++
		seenPaths[d.path] = true
		addedKeywordItems++
		result.Items = append(result.Items, RecallItem{
			Source:  "diary_keyword",
			Date:    d.date,
			Path:    d.path,
			Excerpt: extractKeywordExcerpt(content, contentLower, keywords, 300),
		})
	}

	result.RecallCount = len(result.Items)
	return result, nil
}

func (m *Manager) collectDiaryFiles() ([]diaryFile, error) {
	entries, err := os.ReadDir(m.memoryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	diaries := make([]diaryFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == memoryFileName || !strings.HasSuffix(name, ".md") {
			continue
		}
		date := strings.TrimSuffix(name, ".md")
		if !isValidDate(date) {
			continue
		}
		diaries = append(diaries, diaryFile{
			date: date,
			path: filepath.Join(m.memoryDir, name),
		})
	}
	return diaries, nil
}

func extractRecallKeywords(query string) []string {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	parts := strings.FieldsFunc(query, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.Is(unicode.Han, r)
	})
	seen := map[string]bool{}
	keywords := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		if utf8RuneLen(p) < 2 {
			continue
		}
		seen[p] = true
		keywords = append(keywords, p)
	}
	return keywords
}

func containsAnyKeyword(contentLower string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}
	return false
}

func extractKeywordExcerpt(content, contentLower string, keywords []string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 280
	}
	if content == "" {
		return ""
	}

	matchIndex := -1
	for _, keyword := range keywords {
		idx := strings.Index(contentLower, keyword)
		if idx >= 0 && (matchIndex == -1 || idx < matchIndex) {
			matchIndex = idx
		}
	}
	if matchIndex < 0 {
		return clipText(content, maxLen)
	}

	start := matchIndex - maxLen/3
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(content) {
		end = len(content)
	}
	snippet := strings.TrimSpace(content[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet += "..."
	}
	return snippet
}

func clipText(content string, maxLen int) string {
	content = strings.TrimSpace(content)
	if maxLen <= 0 || len(content) <= maxLen {
		return content
	}
	return strings.TrimSpace(content[:maxLen]) + "..."
}

// utf8RuneLen returns the number of characters in a string.
// Optimized to use utf8.RuneCountInString to avoid O(N) memory allocation from casting to a rune slice.
func utf8RuneLen(text string) int {
	return utf8.RuneCountInString(text)
}

// isValidDate performs a fast, zero-allocation check to verify a string matches the YYYY-MM-DD format.
// It skips the expensive leap-year and bounds validation of time.Parse as diary filenames are machine-generated.
func isValidDate(s string) bool {
	if len(s) != 10 {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	for i := 0; i < 10; i++ {
		if i == 4 || i == 7 {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
