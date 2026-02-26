package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	memoryDirName  = "memory"
	memoryFileName = "MEMORY.md"
)

type DiaryEntry struct {
	Date    string
	Path    string
	Content string
}

// RecallItem is one recalled memory fragment with source attribution.
type RecallItem struct {
	Source  string
	Date    string
	Path    string
	Excerpt string
}

// RecallResult summarizes context recall quality and provenance.
type RecallResult struct {
	Query       string
	RecallCount int
	SourceHits  map[string]int
	Items       []RecallItem
}

type diaryFile struct {
	date string
	path string
}

type Manager struct {
	workspacePath string
	memoryDir     string
	memoryFile    string
}

func NewManager(workspacePath string) *Manager {
	memoryDir := filepath.Join(workspacePath, memoryDirName)
	return &Manager{
		workspacePath: workspacePath,
		memoryDir:     memoryDir,
		memoryFile:    filepath.Join(memoryDir, memoryFileName),
	}
}

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

func (m *Manager) ReadLongTerm() (string, error) {
	if err := m.Ensure(); err != nil {
		return "", err
	}
	data, err := os.ReadFile(m.memoryFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (m *Manager) WriteLongTerm(content string) error {
	if err := m.Ensure(); err != nil {
		return err
	}
	return os.WriteFile(m.memoryFile, []byte(strings.TrimSpace(content)), 0644)
}

func (m *Manager) AppendDiary(entry string) (string, error) {
	return m.AppendDiaryAt(time.Now(), entry)
}

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

func (m *Manager) ReadRecentDiaries(limit int) ([]DiaryEntry, error) {
	if limit <= 0 {
		limit = 3
	}
	if err := m.Ensure(); err != nil {
		return nil, err
	}
	diaries, err := m.collectDiaryFiles()
	if err != nil {
		return nil, err
	}

	sort.Slice(diaries, func(i, j int) bool {
		return diaries[i].date > diaries[j].date
	})
	if len(diaries) > limit {
		diaries = diaries[:limit]
	}
	sort.Slice(diaries, func(i, j int) bool {
		return diaries[i].date < diaries[j].date
	})

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

// RecallContext selects context fragments using recent-first + keyword-hit strategy.
func (m *Manager) RecallContext(query string, recentLimit, keywordLimit int) (RecallResult, error) {
	if recentLimit <= 0 {
		recentLimit = 3
	}
	if keywordLimit <= 0 {
		keywordLimit = 3
	}
	if err := m.Ensure(); err != nil {
		return RecallResult{}, err
	}

	result := RecallResult{
		Query:      strings.TrimSpace(query),
		SourceHits: map[string]int{},
		Items:      make([]RecallItem, 0, recentLimit+keywordLimit+1),
	}
	seenPaths := map[string]bool{}

	// 1. Collect all diary files once
	diaries, err := m.collectDiaryFiles()
	if err != nil {
		return RecallResult{}, err
	}
	sort.Slice(diaries, func(i, j int) bool {
		return diaries[i].date > diaries[j].date // Newest first
	})

	// 2. Process recent entries directly from the sorted list
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

	// 3. Process long-term memory
	longTerm, err := m.ReadLongTerm()
	if err != nil {
		return RecallResult{}, err
	}
	if strings.TrimSpace(longTerm) != "" && containsAnyKeyword(longTerm, keywords) {
		result.SourceHits["long_term"]++
		result.Items = append(result.Items, RecallItem{
			Source:  "long_term",
			Date:    "",
			Path:    m.memoryFile,
			Excerpt: extractKeywordExcerpt(longTerm, keywords, 380),
		})
	}

	// 4. Process keyword search on remaining diaries
	addedKeywordItems := 0
	for _, d := range diaries {
		if addedKeywordItems >= keywordLimit {
			break
		}

		// Optimization: Skip file read if already processed as "recent"
		if seenPaths[d.path] {
			continue
		}

		contentRaw, readErr := os.ReadFile(d.path)
		if readErr != nil {
			continue
		}
		content := strings.TrimSpace(string(contentRaw))
		if content == "" || !containsAnyKeyword(content, keywords) {
			continue
		}

		result.SourceHits["diary_keyword"]++
		seenPaths[d.path] = true
		addedKeywordItems++
		result.Items = append(result.Items, RecallItem{
			Source:  "diary_keyword",
			Date:    d.date,
			Path:    d.path,
			Excerpt: extractKeywordExcerpt(content, keywords, 300),
		})
	}

	result.RecallCount = len(result.Items)
	return result, nil
}

func (m *Manager) collectDiaryFiles() ([]diaryFile, error) {
	entries, err := os.ReadDir(m.memoryDir)
	if err != nil {
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
		if _, err := time.Parse("2006-01-02", date); err != nil {
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

func containsAnyKeyword(content string, keywords []string) bool {
	contentLower := strings.ToLower(content)
	for _, keyword := range keywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}
	return false
}

func extractKeywordExcerpt(content string, keywords []string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 280
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	contentLower := strings.ToLower(content)
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

func utf8RuneLen(text string) int {
	return len([]rune(text))
}
