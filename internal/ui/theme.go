package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme is the small visual vocabulary used by all commands. It intentionally
// stays close to the terminal's own palette so it works in light and dark
// themes.
type Theme struct {
	Title       lipgloss.Style
	Section     lipgloss.Style
	Label       lipgloss.Style
	Muted       lipgloss.Style
	Success     lipgloss.Style
	Warning     lipgloss.Style
	Error       lipgloss.Style
	Accent      lipgloss.Style
	Border      lipgloss.Style
	TableHeader lipgloss.Style
	TableCell   lipgloss.Style
}

// Symbols keeps Unicode out of plain output where it can break scripts.
type Symbols struct {
	Success string
	Failure string
	Warning string
	Arrow   string
	Bullet  string
}

func newTheme(noColor bool) Theme {
	accent := lipgloss.Color("#7D56F4")
	success := lipgloss.Color("#2E7D32")
	warning := lipgloss.Color("#B26A00")
	failure := lipgloss.Color("#C62828")
	muted := lipgloss.Color("#6B7280")

	style := func(useColor bool, foreground color.Color) lipgloss.Style {
		s := lipgloss.NewStyle()
		if useColor {
			s = s.Foreground(foreground)
		}
		return s
	}

	theme := Theme{
		Title:       style(!noColor, accent).Bold(true),
		Section:     style(!noColor, accent).Bold(true),
		Label:       lipgloss.NewStyle().Bold(true),
		Muted:       style(!noColor, muted),
		Success:     style(!noColor, success),
		Warning:     style(!noColor, warning),
		Error:       style(!noColor, failure),
		Accent:      style(!noColor, accent),
		Border:      style(!noColor, muted),
		TableHeader: lipgloss.NewStyle().Bold(true),
		TableCell:   lipgloss.NewStyle(),
	}
	return theme
}

func newSymbols(plain bool) Symbols {
	if plain {
		return Symbols{
			Success: "ok",
			Failure: "x",
			Warning: "!",
			Arrow:   "->",
			Bullet:  "-",
		}
	}
	return Symbols{
		Success: "✓",
		Failure: "✗",
		Warning: "!",
		Arrow:   "→",
		Bullet:  "•",
	}
}
