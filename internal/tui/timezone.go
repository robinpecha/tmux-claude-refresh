package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// tzVisibleCount is how many timezone rows the picker shows at once.
const tzVisibleCount = 12

// cyan renders text in the accent cyan foreground.
var cyan = lipgloss.NewStyle().Foreground(accentCyan)

// filteredTimezones returns the subset of ianaTimezones whose names contain
// filter (case-insensitive). Empty filter returns the full list.
func filteredTimezones(filter string) []string {
	f := strings.ToLower(filter)
	if f == "" {
		return ianaTimezones
	}
	out := make([]string, 0, 64)
	for _, z := range ianaTimezones {
		if strings.Contains(strings.ToLower(z), f) {
			out = append(out, z)
		}
	}
	return out
}

// renderTZPicker renders the timezone picker modal. The selected row is
// highlighted and kept centered in the visible window.
func (m Model) renderTZPicker() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentCyan).
		Padding(1, 2).
		Width(m.width - 4)

	titleLine := titleStyle.Render("tmux-claude-refresh") + " " +
		versionStyle.Render(fmt.Sprintf("v%s", m.version))

	heading := lipgloss.NewStyle().Bold(true).Foreground(accentCyan).Render("TIMEZONE") + "\n" +
		dimTextStyle.Render("Reset times display in this timezone. Saved to ~/.config/tmux-claude-refresh/config.")

	// Filter input with a block cursor.
	filterStyled := lipgloss.NewStyle().Bold(true).Foreground(brightWhite).Render(m.tzFilter)
	input := cyan.Render("> ") + filterStyled + lipgloss.NewStyle().Foreground(accentCyan).Render("▏")

	matches := filteredTimezones(m.tzFilter)

	var list string
	if len(matches) == 0 {
		list = dimTextStyle.Render("  no timezones match")
	} else {
		// Clamp selection into range.
		if m.tzIndex < 0 {
			m.tzIndex = 0
		}
		if m.tzIndex >= len(matches) {
			m.tzIndex = len(matches) - 1
		}
		// Centered visible window.
		half := tzVisibleCount / 2
		start := m.tzIndex - half
		if start < 0 {
			start = 0
		}
		end := start + tzVisibleCount
		if end > len(matches) {
			end = len(matches)
			start = end - tzVisibleCount
			if start < 0 {
				start = 0
			}
		}
		var rows []string
		for i := start; i < end; i++ {
			prefix := "  "
			text := matches[i]
			row := prefix + text
			if i == m.tzIndex {
				row = cyan.Render("❯ ") + lipgloss.NewStyle().Bold(true).Foreground(brightWhite).Render(text)
			} else {
				row = "  " + dimTextStyle.Render(text)
			}
			rows = append(rows, row)
		}
		// Scroll indicators.
		lead := ""
		if start > 0 {
			lead = dimTextStyle.Render("  ↑ more") + "\n"
		}
		trail := ""
		if end < len(matches) {
			trail = "\n" + dimTextStyle.Render("  ↓ more")
		}
		list = lead + strings.Join(rows, "\n") + trail
	}

	var errLine string
	if m.tzError != "" {
		errLine = "\n" + errorStyle.Render(m.tzError)
	}

	footer := dimTextStyle.Render("enter select • esc cancel • type to filter • ↑↓ navigate")

	content := heading + "\n\n" + input + "\n\n" + list + errLine + "\n\n" + footer

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		boxStyle.Render(titleLine+"\n\n"+content),
		"  "+footer,
	)
}
