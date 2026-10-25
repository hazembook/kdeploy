// Package ui owns the presentation and interaction layer for kdeploy.
//
// It deliberately knows nothing about VMs, config, or command policy. Commands
// decide what to ask and what to do; this package decides how it looks, where
// it goes, and whether an interactive terminal is available.
package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
)

var (
	// ErrNonInteractive is returned when a command needs a prompt but stdin or
	// stderr is not a terminal.
	ErrNonInteractive = errors.New("interactive terminal required")

	// ErrCancelled is returned when the user aborts a form with Ctrl+C.
	ErrCancelled = errors.New("cancelled")
)

// Options is the test seam for UI construction. Zero values use os.Stdin,
// os.Stdout, os.Stderr, os.Environ(), and terminal detection.
type Options struct {
	In          io.Reader
	Out         io.Writer
	Err         io.Writer
	Env         []string
	Interactive *bool
	Width       int
}

// UI is the shared presentation object for a command invocation.
type UI struct {
	In          io.Reader
	Out         io.Writer
	Err         io.Writer
	Interactive bool
	Plain       bool
	NoColor     bool
	Width       int

	Theme   Theme
	Symbols Symbols

	inRaw  io.Reader
	errRaw io.Writer
}

// New constructs a UI with production defaults.
func New() *UI {
	return NewWithOptions(Options{})
}

// NewWithOptions constructs a UI with an explicit test seam.
func NewWithOptions(opts Options) *UI {
	in := opts.In
	if in == nil {
		in = os.Stdin
	}
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Err
	if errOut == nil {
		errOut = os.Stderr
	}
	env := opts.Env
	if env == nil {
		env = os.Environ()
	}

	interactive := false
	if opts.Interactive != nil {
		interactive = *opts.Interactive
	} else {
		inFile, inOK := in.(*os.File)
		errFile, errOK := errOut.(*os.File)
		interactive = inOK && errOK &&
			term.IsTerminal(inFile.Fd()) &&
			term.IsTerminal(errFile.Fd())
	}

	width := opts.Width
	if width <= 0 {
		width = detectWidth(out)
	}
	if width <= 0 {
		width = 80
	}

	plain := !interactive || strings.EqualFold(envValue(env, "TERM"), "dumb")
	noColor := plain || envValue(env, "NO_COLOR") != ""

	return &UI{
		In:          in,
		Out:         colorprofile.NewWriter(out, env),
		Err:         colorprofile.NewWriter(errOut, env),
		Interactive: interactive,
		Plain:       plain,
		NoColor:     noColor,
		Width:       width,
		Theme:       newTheme(noColor),
		Symbols:     newSymbols(plain),
		inRaw:       in,
		errRaw:      errOut,
	}
}

// CanPrompt reports whether a form can be shown.
func (u *UI) CanPrompt() bool {
	return u.Interactive
}

// RequireInteractive returns an actionable error when a prompt cannot be used.
func (u *UI) RequireInteractive(action string) error {
	if u.Interactive {
		return nil
	}
	return fmt.Errorf("%w: %s; use --yes for non-interactive mode", ErrNonInteractive, action)
}

// MaskSecret hides a secret in display output without pretending it is absent.
func (u *UI) MaskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	return "********"
}

func detectWidth(out io.Writer) int {
	file, ok := out.(*os.File)
	if !ok || !term.IsTerminal(file.Fd()) {
		return 80
	}
	width, _, err := term.GetSize(file.Fd())
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(env[i], prefix) {
			return strings.TrimPrefix(env[i], prefix)
		}
	}
	return ""
}
