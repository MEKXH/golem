## 2026-02-28 - Tabular Output Styling
**Learning:** When adding colors to tabular CLI data, using standard padding (`fmt.Printf("%-10s")`) breaks alignment because ANSI color codes are counted as characters. Relying on `fmt.Printf` for colorful tabular output leads to jagged columns.
**Action:** Replaced `fmt.Printf` with `lipgloss.JoinHorizontal` and `lipgloss.Style.Width()` to calculate lengths independently of invisible escape sequences, ensuring perfect alignment.

## 2026-03-05 - TUI Spacer Right Alignment
**Learning:** When trying to align UI elements to opposite ends of a terminal row using `lipgloss` (e.g. left-aligned keybindings and a right-aligned "Scrolled Up" indicator), simply appending them via `JoinHorizontal` bunches them together on the left.
**Action:** Use a dynamic spacer by calculating `availableWidth - lipgloss.Width(leftElements) - lipgloss.Width(rightElement)` and inserting `strings.Repeat(" ", spacerWidth)` using `lipgloss.JoinHorizontal` to cleanly push the element to the right edge.

## 2026-03-07 - Empty State Actions
**Learning:** Empty states in CLI commands (e.g., "No skills installed", "No scheduled jobs") can leave users frustrated because they don't immediately know how to resolve the state. Providing actionable guidance directly in the empty state message drastically improves discoverability.
**Action:** When printing an empty state message, always include a helpful call-to-action or suggest the exact command needed to populate the state (e.g., "No scheduled jobs. Use 'golem cron add' to create one.").
