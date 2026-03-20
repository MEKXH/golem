package geopipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var matcherStopWords = map[string]bool{
	"analyze":    true,
	"analysis":   true,
	"another":    true,
	"area":       true,
	"new":        true,
	"with":       true,
	"for":        true,
	"into":       true,
	"from":       true,
	"summarize":  true,
	"preprocess": true,
	"process":    true,
}

// Match describes one ranked learned geo pipeline match.
type Match struct {
	ID        string `json:"id"`
	Goal      string `json:"goal"`
	CreatedAt string `json:"created_at"`
	Steps     []Step `json:"steps"`
	Score     int    `json:"score"`
}

// Matcher ranks learned geo pipelines against a freeform goal.
type Matcher struct {
	workspacePath string
}

// NewMatcher creates a matcher for workspace learned geo pipelines.
func NewMatcher(workspacePath string) *Matcher {
	return &Matcher{workspacePath: strings.TrimSpace(workspacePath)}
}

// Find returns the top matching learned geo pipelines for the given goal.
func (m *Matcher) Find(goal string, limit int) ([]Match, error) {
	records, err := m.loadRecords()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}

	goalTokens := tokenize(goal)
	matches := make([]Match, 0, len(records))
	type rankMeta struct {
		match Match
		extra int
	}
	ranked := make([]rankMeta, 0, len(records))
	for _, record := range records {
		score, extra := scoreRecord(record, goalTokens)
		if len(goalTokens) > 0 && score == 0 {
			continue
		}
		ranked = append(ranked, rankMeta{match: Match{
			ID:        record.ID,
			Goal:      record.Goal,
			CreatedAt: record.CreatedAt,
			Steps:     append([]Step(nil), record.Steps...),
			Score:     score,
		}, extra: extra})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].match.Score != ranked[j].match.Score {
			return ranked[i].match.Score > ranked[j].match.Score
		}
		if ranked[i].extra != ranked[j].extra {
			return ranked[i].extra < ranked[j].extra
		}
		if ranked[i].match.CreatedAt != ranked[j].match.CreatedAt {
			return ranked[i].match.CreatedAt > ranked[j].match.CreatedAt
		}
		return ranked[i].match.Goal < ranked[j].match.Goal
	})

	for _, item := range ranked {
		matches = append(matches, item.match)
	}
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func (m *Matcher) loadRecords() ([]Record, error) {
	dir := filepath.Join(m.workspacePath, "pipelines", "geo")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read learned pipeline dir: %w", err)
	}

	records := make([]Record, 0, len(entries))
	recorder := NewRecorder(m.workspacePath)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		record, err := recorder.load(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		if strings.TrimSpace(record.Goal) == "" {
			continue
		}
		records = append(records, record)
	}
	return records, nil
}

var tokenizeReplacer = strings.NewReplacer(",", " ", ".", " ", "_", " ", "-", " ", "/", " ", "(", " ", ")", " ")

func tokenize(input string) []string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return nil
	}
	input = tokenizeReplacer.Replace(input)
	fields := strings.Fields(input)
	seen := make(map[string]bool, len(fields))
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) < 2 || seen[field] || matcherStopWords[field] {
			continue
		}
		seen[field] = true
		tokens = append(tokens, field)
	}
	return tokens
}

func scoreRecord(record Record, goalTokens []string) (int, int) {
	if len(goalTokens) == 0 {
		return 1, 0
	}

	recordTokens := tokenize(record.Goal)
	goalSet := make(map[string]bool, len(goalTokens))
	for _, token := range goalTokens {
		goalSet[token] = true
	}

	score := 0
	extra := 0
	for _, token := range recordTokens {
		if goalSet[token] {
			score++
			continue
		}
		extra++
	}
	return score, extra
}
