package ui

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Heading prints a primary result heading to stdout.
func (u *UI) Heading(text string) {
	fmt.Fprintln(u.Out, u.Theme.Title.Render(text))
}

// Section prints a progress/section label to stderr.
func (u *UI) Section(text string) {
	fmt.Fprintln(u.Err, u.Theme.Section.Render(text))
}

// KV prints aligned key/value rows to stdout.
func (u *UI) KV(pairs ...string) {
	if len(pairs)%2 != 0 {
		pairs = append(pairs, "")
	}
	width := 0
	for i := 0; i < len(pairs); i += 2 {
		if w := len(pairs[i]); w > width {
			width = w
		}
	}
	for i := 0; i < len(pairs); i += 2 {
		label := pairs[i] + ":"
		value := pairs[i+1]
		if u.Plain {
			fmt.Fprintf(u.Out, "%-*s  %s\n", width+1, label, value)
			continue
		}
		fmt.Fprintf(u.Out, "%s  %s\n", u.Theme.Label.Render(fmt.Sprintf("%-*s", width+1, label)), value)
	}
}

// Summary prints a heading followed by a table.
func (u *UI) Summary(title string, headers []string, rows [][]string) {
	u.Heading(title)
	u.Table(headers, rows)
}

// Success prints a success status to stderr.
func (u *UI) Success(format string, args ...any) {
	u.status(u.Theme.Success.Render(u.Symbols.Success), format, args...)
}

// Warn prints a warning status to stderr.
func (u *UI) Warn(format string, args ...any) {
	u.status(u.Theme.Warning.Render(u.Symbols.Warning), format, args...)
}

// Status prints a neutral status to stderr.
func (u *UI) Status(format string, args ...any) {
	u.status(u.Theme.Accent.Render(u.Symbols.Arrow), format, args...)
}

// Muted prints a secondary line to stderr.
func (u *UI) Muted(format string, args ...any) {
	fmt.Fprintln(u.Err, u.Theme.Muted.Render(fmt.Sprintf(format, args...)))
}

func (u *UI) status(symbol, format string, args ...any) {
	fmt.Fprintf(u.Err, "%s %s\n", symbol, fmt.Sprintf(format, args...))
}

// Print prints a plain result line to stdout.
func (u *UI) Print(format string, args ...any) {
	fmt.Fprintf(u.Out, format, args...)
}

// Println prints a plain result line to stdout.
func (u *UI) Println(args ...any) {
	fmt.Fprintln(u.Out, args...)
}

// PlainTable writes a script-friendly tab-separated table.
func (u *UI) PlainTable(out io.Writer, headers []string, rows [][]string) {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}
