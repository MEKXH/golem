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

// OutcomeCount returns the number of recorded success/failure outcomes.
func (e TelemetryEntry) OutcomeCount() int {
	return e.Success + e.Failure
}

// HasOutcomeData reports whether the entry has any success/failure observations.
func (e TelemetryEntry) HasOutcomeData() bool {
	return e.OutcomeCount() > 0
}

// SuccessRatio returns a coarse success ratio across recorded outcomes.
func (e TelemetryEntry) SuccessRatio() float64 {
	outcomes := e.OutcomeCount()
	if outcomes == 0 {
		return 0
	}
	return float64(e.Success) / float64(outcomes)
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

var skillQueryReplacer = strings.NewReplacer("-", " ", "_", " ", "/", " ")

func normalizeSkillQueryText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = skillQueryReplacer.Replace(text)
	return strings.Join(strings.Fields(text), " ")
}
