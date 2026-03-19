package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TelemetryEntry stores coarse-grained usage counters for one skill.
type TelemetryEntry struct {
	Shown    int `json:"shown"`
	Selected int `json:"selected"`
	Success  int `json:"success"`
	Failure  int `json:"failure"`
}

// TelemetrySnapshot is the persisted workspace skill telemetry state.
type TelemetrySnapshot struct {
	Path   string                    `json:"-"`
	Skills map[string]TelemetryEntry `json:"skills"`
}

// TelemetryRecorder persists skill telemetry in the workspace state directory.
type TelemetryRecorder struct {
	workspacePath string
}

// NewTelemetryRecorder creates a recorder for workspace skill telemetry.
func NewTelemetryRecorder(workspacePath string) *TelemetryRecorder {
	return &TelemetryRecorder{workspacePath: strings.TrimSpace(workspacePath)}
}

func (r *TelemetryRecorder) RecordShown(skills []SkillInfo) error {
	seen := make(map[string]bool, len(skills))
	return r.update(func(snapshot *TelemetrySnapshot) {
		for _, skill := range skills {
			name := strings.TrimSpace(skill.Name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			entry := snapshot.Skills[name]
			entry.Shown++
			snapshot.Skills[name] = entry
		}
	})
}

func (r *TelemetryRecorder) RecordSelected(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	return r.update(func(snapshot *TelemetrySnapshot) {
		entry := snapshot.Skills[name]
		entry.Selected++
		snapshot.Skills[name] = entry
	})
}

func (r *TelemetryRecorder) RecordOutcome(name string, success bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	return r.update(func(snapshot *TelemetrySnapshot) {
		entry := snapshot.Skills[name]
		if success {
			entry.Success++
		} else {
			entry.Failure++
		}
		snapshot.Skills[name] = entry
	})
}

func (r *TelemetryRecorder) Load() (TelemetrySnapshot, error) {
	path := r.path()
	if strings.TrimSpace(path) == "" {
		return TelemetrySnapshot{Skills: map[string]TelemetryEntry{}}, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TelemetrySnapshot{Path: path, Skills: map[string]TelemetryEntry{}}, nil
		}
		return TelemetrySnapshot{}, fmt.Errorf("read skill telemetry: %w", err)
	}

	var snapshot TelemetrySnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		return TelemetrySnapshot{}, fmt.Errorf("parse skill telemetry: %w", err)
	}
	if snapshot.Skills == nil {
		snapshot.Skills = map[string]TelemetryEntry{}
	}
	snapshot.Path = path
	return snapshot, nil
}

func (r *TelemetryRecorder) update(fn func(snapshot *TelemetrySnapshot)) error {
	snapshot, err := r.Load()
	if err != nil {
		return err
	}
	if snapshot.Skills == nil {
		snapshot.Skills = map[string]TelemetryEntry{}
	}
	fn(&snapshot)
	return r.save(snapshot)
}

func (r *TelemetryRecorder) save(snapshot TelemetrySnapshot) error {
	path := r.path()
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create skill telemetry dir: %w", err)
	}
	body, err := json.MarshalIndent(TelemetrySnapshot{Skills: snapshot.Skills}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal skill telemetry: %w", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write skill telemetry: %w", err)
	}
	return nil
}

func (r *TelemetryRecorder) path() string {
	if strings.TrimSpace(r.workspacePath) == "" {
		return ""
	}
	return filepath.Join(r.workspacePath, "state", "skill_telemetry.json")
}

// SelectSkillsForQuery returns explicitly referenced workspace skills for the given query.
func SelectSkillsForQuery(skills []SkillInfo, query string) []SkillInfo {
	normalizedQuery := normalizeSkillQueryText(query)
	if normalizedQuery == "" {
		return nil
	}

	selected := make([]SkillInfo, 0)
	seen := make(map[string]bool)
	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" || seen[name] || skill.Source != "workspace" {
			continue
		}
		for _, alias := range skillQueryAliases(name) {
			if alias != "" && strings.Contains(normalizedQuery, alias) {
				selected = append(selected, skill)
				seen[name] = true
				break
			}
		}
	}
	return selected
}

func skillQueryAliases(name string) []string {
	alias := normalizeSkillQueryText(name)
	if alias == "" {
		return nil
	}
	return []string{alias}
}

func normalizeSkillQueryText(text string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ", "/", " ")
	text = strings.ToLower(strings.TrimSpace(text))
	text = replacer.Replace(text)
	return strings.Join(strings.Fields(text), " ")
}
