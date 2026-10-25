package ui

import (
	"context"
	"errors"
	"fmt"

	"charm.land/huh/v2"
)

// RunForm runs a Huh form with the shared input/output theme and maps Ctrl+C
// cancellation to ErrCancelled.
func (u *UI) RunForm(ctx context.Context, form *huh.Form) error {
	if err := u.RequireInteractive("this prompt"); err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	width := u.Width
	if width > 100 {
		width = 100
	}
	if width < 20 {
		width = 20
	}

	form.
		WithInput(u.inRaw).
		WithOutput(u.errRaw).
		WithTheme(huh.ThemeFunc(huh.ThemeCharm)).
		WithWidth(width)

	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return ErrCancelled
		}
		return fmt.Errorf("prompt: %w", err)
	}
	return nil
}

// Confirm is a small helper for destructive confirmations that default to No.
func (u *UI) Confirm(ctx context.Context, title string, value *bool) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Yes").
				Negative("No").
				Value(value),
		),
	)
	return u.RunForm(ctx, form)
}
