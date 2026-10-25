package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// Table renders a styled table on a terminal and a stable tab-separated table
// when output is piped, redirected, or otherwise plain.
func (u *UI) Table(headers []string, rows [][]string) {
	if u.Plain {
		u.PlainTable(u.Out, headers, rows)
		return
	}

	width := u.Width
	if width > 120 {
		width = 120
	}
	if width < 20 {
		width = 20
	}

	t := table.New().
		Headers(headers...).
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(u.Theme.Border).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return u.Theme.TableHeader
			}
			return u.Theme.TableCell
		}).
		Width(width)

	fmt.Fprintln(u.Out, t.Render())
}
