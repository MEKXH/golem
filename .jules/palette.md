## 2026-02-28 - Tabular Output Styling
**Learning:** When adding colors to tabular CLI data, using standard padding (`fmt.Printf("%-10s")`) breaks alignment because ANSI color codes are counted as characters. Relying on `fmt.Printf` for colorful tabular output leads to jagged columns.
**Action:** Replaced `fmt.Printf` with `lipgloss.JoinHorizontal` and `lipgloss.Style.Width()` to calculate lengths independently of invisible escape sequences, ensuring perfect alignment.
