package skills

import "sort"

// ReportEntry summarizes telemetry for one skill.
type ReportEntry struct {
	Name           string
	Shown          int
	Selected       int
	Success        int
	Failure        int
	SuccessRatio   float64
	HasOutcomeData bool
}

// TelemetryReport is a deterministic snapshot view over persisted skill telemetry.
type TelemetryReport struct {
	Entries []ReportEntry
}

// BuildTelemetryReport converts a telemetry snapshot into a sorted report view.
func BuildTelemetryReport(snapshot TelemetrySnapshot) TelemetryReport {
	entries := make([]ReportEntry, 0, len(snapshot.Skills))
	for name, entry := range snapshot.Skills {
		entries = append(entries, ReportEntry{
			Name:           name,
			Shown:          entry.Shown,
			Selected:       entry.Selected,
			Success:        entry.Success,
			Failure:        entry.Failure,
			SuccessRatio:   entry.SuccessRatio(),
			HasOutcomeData: entry.HasOutcomeData(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]
		if left.HasOutcomeData != right.HasOutcomeData {
			return left.HasOutcomeData
		}
		if left.HasOutcomeData && right.HasOutcomeData {
			if left.SuccessRatio != right.SuccessRatio {
				return left.SuccessRatio < right.SuccessRatio
			}
			leftOutcomes := left.Success + left.Failure
			rightOutcomes := right.Success + right.Failure
			if leftOutcomes != rightOutcomes {
				return leftOutcomes > rightOutcomes
			}
		}
		if left.Selected != right.Selected {
			return left.Selected > right.Selected
		}
		if left.Shown != right.Shown {
			return left.Shown > right.Shown
		}
		return left.Name < right.Name
	})

	return TelemetryReport{Entries: entries}
}
