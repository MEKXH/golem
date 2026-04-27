package geocodebook

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Variable describes a codebook template variable.
type Variable struct {
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Required    bool   `yaml:"required" json:"required"`
	Default     string `yaml:"default" json:"default"`
}

// Pattern describes a reusable spatial SQL pattern.
type Pattern struct {
	Name        string              `yaml:"name" json:"name"`
	Description string              `yaml:"description" json:"description"`
	Tags        []string            `yaml:"tags" json:"tags"`
	Template    string              `yaml:"template" json:"template"`
	Variables   map[string]Variable `yaml:"variables" json:"variables"`
	Verified    bool                `yaml:"verified" json:"verified"`
	SuccessRate float64             `yaml:"success_rate" json:"success_rate"`
	Source      string              `yaml:"-" json:"source"`
}

type document struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Patterns    []Pattern `yaml:"patterns"`
}

// Match describes a ranked pattern result for an intent lookup.
type Match struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Verified    bool     `json:"verified"`
	SuccessRate float64  `json:"success_rate"`
	Score       int      `json:"score"`
	Source      string   `json:"source"`
}

// RenderedPattern is a fully rendered SQL pattern.
type RenderedPattern struct {
	Pattern   string            `json:"pattern"`
	SQL       string            `json:"sql"`
	Variables map[string]string `json:"variables"`
	Verified  bool              `json:"verified"`
	Source    string            `json:"source"`
}

// Loader loads spatial SQL codebook patterns from <workspace>/geo-codebook.
type Loader struct {
	root string
}

// NewLoader creates a codebook loader for the workspace.
func NewLoader(workspacePath string) *Loader {
	return &Loader{root: filepath.Join(workspacePath, "geo-codebook")}
}

// ListPatterns ranks patterns against a freeform intent.
func (l *Loader) ListPatterns(intent string, limit int) ([]Match, error) {
	patterns, err := l.loadPatterns()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	intentTokens := tokenize(intent)
	matches := make([]Match, 0, len(patterns))
	for _, pattern := range patterns {
		score := patternScore(pattern, intentTokens)
		if len(intentTokens) > 0 && score == 0 {
			continue
		}
		matches = append(matches, Match{
			Name:        pattern.Name,
			Description: pattern.Description,
			Tags:        append([]string(nil), pattern.Tags...),
			Verified:    pattern.Verified,
			SuccessRate: pattern.SuccessRate,
			Score:       score,
			Source:      pattern.Source,
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].Verified != matches[j].Verified {
			return matches[i].Verified
		}
		if matches[i].SuccessRate != matches[j].SuccessRate {
			return matches[i].SuccessRate > matches[j].SuccessRate
		}
		return matches[i].Name < matches[j].Name
	})

	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

// RenderPattern renders a named SQL pattern with supplied variables.
func (l *Loader) RenderPattern(name string, values map[string]string) (*RenderedPattern, error) {
	patterns, err := l.loadPatterns()
	if err != nil {
		return nil, err
	}

	var found *Pattern
	for i := range patterns {
		if patterns[i].Name == name {
			found = &patterns[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("codebook pattern not found: %s", name)
	}

	resolved := make(map[string]string, len(found.Variables))
	replacements := make([]string, 0, len(found.Variables)*2)
	for key, spec := range found.Variables {
		value := strings.TrimSpace(values[key])
		if value == "" {
			value = strings.TrimSpace(spec.Default)
		}
		if value == "" && spec.Required {
			return nil, fmt.Errorf("missing required variable %q", key)
		}
		resolved[key] = value
		replacements = append(replacements, "{{"+key+"}}", value)
	}

	sql := strings.NewReplacer(replacements...).Replace(found.Template)
	if strings.Contains(sql, "{{") {
		return nil, fmt.Errorf("unresolved placeholders remain in rendered SQL")
	}

	return &RenderedPattern{
		Pattern:   found.Name,
		SQL:       strings.TrimSpace(sql),
		Variables: resolved,
		Verified:  found.Verified,
		Source:    found.Source,
	}, nil
}

// BuildSummary returns a concise prompt-ready summary of available codebook patterns.
func (l *Loader) BuildSummary() (string, error) {
	patterns, err := l.loadPatterns()
	if err != nil {
		return "", err
	}
	if len(patterns) == 0 {
		return "", nil
	}

	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].Verified != patterns[j].Verified {
			return patterns[i].Verified
		}
		if patterns[i].SuccessRate != patterns[j].SuccessRate {
			return patterns[i].SuccessRate > patterns[j].SuccessRate
		}
		return patterns[i].Name < patterns[j].Name
	})

	var sb strings.Builder
	sb.WriteString("## Spatial SQL Codebook\n\n")
	sb.WriteString("Prefer these verified spatial SQL patterns before composing SQL from scratch.\n")
	for _, pattern := range patterns {
		sb.WriteString(fmt.Sprintf("- `%s`: %s", pattern.Name, pattern.Description))
		if len(pattern.Tags) > 0 {
			sb.WriteString(fmt.Sprintf(" (tags: %s)", strings.Join(pattern.Tags, ", ")))
		}
		if pattern.Verified {
			sb.WriteString(fmt.Sprintf(" [verified %.2f]", pattern.SuccessRate))
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

func (l *Loader) loadPatterns() ([]Pattern, error) {
	entries, err := os.ReadDir(l.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read geo-codebook directory: %w", err)
	}

	patterns := make([]Pattern, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(l.root, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read codebook file %s: %w", path, err)
		}

		var doc document
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse codebook file %s: %w", path, err)
		}

		for _, pattern := range doc.Patterns {
			if strings.TrimSpace(pattern.Name) == "" || strings.TrimSpace(pattern.Template) == "" {
				continue
			}
			pattern.Source = path
			patterns = append(patterns, pattern)
		}
	}

	return patterns, nil
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
		if len(field) < 2 || seen[field] {
			continue
		}
		seen[field] = true
		tokens = append(tokens, field)
	}
	return tokens
}

func patternScore(pattern Pattern, intentTokens []string) int {
	if len(intentTokens) == 0 {
		return 1
	}

	haystack := strings.ToLower(strings.Join(append([]string{pattern.Name, pattern.Description}, pattern.Tags...), " "))
	score := 0
	for _, token := range intentTokens {
		if strings.Contains(haystack, token) {
			score++
		}
	}
	return score
}
