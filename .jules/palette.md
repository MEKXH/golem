## 2026-02-28 - Tabular Output Styling
**Learning:** When adding colors to tabular CLI data, using standard padding (`fmt.Printf("%-10s")`) breaks alignment because ANSI color codes are counted as characters. Relying on `fmt.Printf` for colorful tabular output leads to jagged columns.
**Action:** Replaced `fmt.Printf` with `lipgloss.JoinHorizontal` and `lipgloss.Style.Width()` to calculate lengths independently of invisible escape sequences, ensuring perfect alignment.

## 2026-03-05 - TUI Spacer Right Alignment
**Learning:** When trying to align UI elements to opposite ends of a terminal row using `lipgloss` (e.g. left-aligned keybindings and a right-aligned "Scrolled Up" indicator), simply appending them via `JoinHorizontal` bunches them together on the left.
**Action:** Use a dynamic spacer by calculating `availableWidth - lipgloss.Width(leftElements) - lipgloss.Width(rightElement)` and inserting `strings.Repeat(" ", spacerWidth)` using `lipgloss.JoinHorizontal` to cleanly push the element to the right edge.

## 2026-03-07 - Empty State Actions
**Learning:** Empty states in CLI commands (e.g., "No skills installed", "No scheduled jobs") can leave users frustrated because they don't immediately know how to resolve the state. Providing actionable guidance directly in the empty state message drastically improves discoverability.
**Action:** When printing an empty state message, always include a helpful call-to-action or suggest the exact command needed to populate the state (e.g., "No scheduled jobs. Use 'golem cron add' to create one.").

## 2026-03-12 - Multibyte String Truncation
**Learning:** When truncating strings for UI display in Go (especially in CLIs where multi-byte characters like emojis or non-English text might appear), slicing by bytes (e.g. `s[:maxLen]`) can produce invalid UTF-8 characters and break rendering, causing ugly symbols in the terminal.
**Action:** Always slice by runes by converting to `[]rune(s)` before calculating length or slicing, ensuring that multibyte characters are kept intact when truncating.

## 2026-03-16 - Destructive Action Confirmation
**Learning:** Destructive CLI commands, such as bulk credential deletion via `golem auth logout` (when executed without specifying a particular provider), risk causing unwanted data loss and frustration if executed accidentally.
**Action:** Implemented a safety prompt (`[y/N]`) that safely aborts on any non-confirming input, alongside a `--yes` (`-y`) flag to bypass the prompt for scripts. This prevents accidental wipes without degrading power-user workflows.

## 2026-03-24 - Dynamic ARIA Labels in Vue
**Learning:** Screen readers cannot infer the purpose of an input element from a Vue dynamic `placeholder` alone. Hardcoding an `aria-label` string disrupts existing `vue-i18n` workflows and creates translation drift.
**Action:** When adding accessible names to form inputs in Vue, bind the `aria-label` attribute directly to the existing translation token (e.g., `:aria-label="consoleCopy.composer.placeholder"`) to maintain accessibility without duplicating translation efforts.