package geopipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Step represents one learned geo tool invocation.
type Step struct {
	Tool     string `yaml:"tool" json:"tool"`
	ArgsJSON string `yaml:"args_json,omitempty" json:"args_json,omitempty"`
}

// Record is the persisted form of a learned geo pipeline.
type Record struct {
	ID        string `yaml:"id" json:"id"`
	Goal      string `yaml:"goal" json:"goal"`
	CreatedAt string `yaml:"created_at" json:"created_at"`
	Steps     []Step `yaml:"steps" json:"steps"`
}

// Recorder persists learned geo pipelines in the workspace.
type Recorder struct {
	workspacePath string
	now           func() string
}

// NewRecorder creates a recorder for workspace geo pipelines.
func NewRecorder(workspacePath string) *Recorder {
	return &Recorder{
		workspacePath: strings.TrimSpace(workspacePath),
		now: func() string {
			return time.Now().UTC().Format("20060102-150405")
		},
	}
}

// Save writes one learned geo pipeline record into the workspace.
func (r *Recorder) Save(goal string, steps []Step) error {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return fmt.Errorf("goal is required")
	}
	if len(steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	dir := filepath.Join(r.workspacePath, "pipelines", "geo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create learned pipeline dir: %w", err)
	}

	timestamp := r.now()
	id := timestamp + "-" + slugify(goal)
	record := Record{
		ID:        id,
		Goal:      goal,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Steps:     steps,
	}
	body, err := yaml.Marshal(&record)
	if err != nil {
		return fmt.Errorf("marshal learned pipeline: %w", err)
	}

	path := filepath.Join(dir, id+".yaml")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write learned pipeline: %w", err)
	}
	return nil
}

// BuildSummary returns a prompt-friendly summary of learned geo pipelines.
func (r *Recorder) BuildSummary() string {
	dir := filepath.Join(r.workspacePath, "pipelines", "geo")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) > 5 {
		names = names[:5]
	}

	var sb strings.Builder
	sb.WriteString("## Learned Geo Pipelines\n\n")
	for _, name := range names {
		record, err := r.load(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		tools := make([]string, 0, len(record.Steps))
		for _, step := range record.Steps {
			if step.Tool != "" {
				tools = append(tools, step.Tool)
			}
		}
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", record.Goal, strings.Join(tools, " -> ")))
	}
	return strings.TrimSpace(sb.String())
}

func (r *Recorder) load(path string) (Record, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var record Record
	if err := yaml.Unmarshal(body, &record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func slugify(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	var b strings.Builder
	lastDash := false
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "pipeline"
	}
	if len(out) > 48 {
		out = strings.Trim(out[:48], "-")
	}
	if out == "" {
		return "pipeline"
	}
	return out
}
