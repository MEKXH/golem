package geopipeline

import "strings"

// ReuseStep describes one step of a learned pipeline as a replay-ready hint.
type ReuseStep struct {
	Tool                 string `json:"tool"`
	ExampleArgsJSON      string `json:"example_args_json,omitempty"`
	NeedsParameterUpdate bool   `json:"needs_parameter_update"`
}

// ReuseCandidate describes one matched learned pipeline as a reusable hint.
type ReuseCandidate struct {
	ID    string      `json:"id"`
	Goal  string      `json:"goal"`
	Score int         `json:"score"`
	Steps []ReuseStep `json:"steps"`
}

// BuildReuseCandidates converts ranked matches into replay-ready reuse hints.
func BuildReuseCandidates(matches []Match) []ReuseCandidate {
	candidates := make([]ReuseCandidate, 0, len(matches))
	for _, match := range matches {
		steps := make([]ReuseStep, 0, len(match.Steps))
		for _, step := range match.Steps {
			exampleArgs := strings.TrimSpace(step.ArgsJSON)
			steps = append(steps, ReuseStep{
				Tool:                 strings.TrimSpace(step.Tool),
				ExampleArgsJSON:      exampleArgs,
				NeedsParameterUpdate: exampleArgs != "",
			})
		}
		candidates = append(candidates, ReuseCandidate{
			ID:    match.ID,
			Goal:  match.Goal,
			Score: match.Score,
			Steps: steps,
		})
	}
	return candidates
}
