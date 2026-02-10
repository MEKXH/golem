package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	entries, err := os.ReadDir(m.memoryDir)
	if err != nil {
		return nil, err
	}

	type diaryFile struct {
		date string
		path string
	}
	var diaries []diaryFile
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
