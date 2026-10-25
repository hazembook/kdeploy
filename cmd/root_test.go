package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/fang"
	"github.com/hazembook/kdeploy/internal/ui"
)

func TestConfigureCommandGroups(t *testing.T) {
	configureCommandGroups()

	want := map[string]string{
		"deploy":   "vm",
		"destroy":  "vm",
		"start":    "vm",
		"shutdown": "vm",
		"stop":     "vm",
		"reboot":   "vm",
		"list":     "inspect",
		"info":     "inspect",
		"ip":       "inspect",
		"console":  "inspect",
		"edit":     "inspect",
		"download": "images",
		"config":   "config",
		"check":    "config",
	}

	for _, command := range rootCmd.Commands() {
		group, ok := want[command.Name()]
		if !ok {
			continue
		}
		if command.GroupID != group {
			t.Errorf("%s group = %q, want %q", command.Name(), command.GroupID, group)
		}
		delete(want, command.Name())
	}
	for name := range want {
		t.Errorf("command %q not found", name)
	}
}

func TestHandleErrorCancelled(t *testing.T) {
	var out bytes.Buffer
	handleError(&out, fang.Styles{}, ui.ErrCancelled)
	if got := out.String(); !strings.Contains(got, "Cancelled.") {
		t.Fatalf("cancellation output = %q", got)
	}
}

func TestHandleErrorPlainWhenPiped(t *testing.T) {
	var out bytes.Buffer
	handleError(&out, fang.Styles{}, ui.ErrNonInteractive)
	got := out.String()
	if !strings.Contains(got, "interactive terminal required") {
		t.Fatalf("plain error output = %q", got)
	}
	if strings.Contains(got, "ERROR") {
		t.Fatalf("plain error output contains a styled header: %q", got)
	}
}
