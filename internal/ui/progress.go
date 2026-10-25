package ui

import (
	"context"

	"charm.land/huh/v2/spinner"
)

// Spin runs a silent operation with a terminal spinner on a TTY. In plain
// mode it simply runs the operation so pipes and logs stay deterministic.
func (u *UI) Spin(ctx context.Context, title string, fn func(context.Context) error) error {
	if u.Plain || fn == nil {
		if fn == nil {
			return nil
		}
		return fn(ctx)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	return spinner.New().
		Type(spinner.Line).
		Title(title).
		WithInput(u.inRaw).
		WithOutput(u.errRaw).
		WithTheme(spinner.ThemeFunc(spinner.ThemeDefault)).
		Context(ctx).
		ActionWithErr(fn).
		Run()
}
