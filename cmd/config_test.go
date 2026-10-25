package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/config"
)

func TestShowConfigMasksPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	cfg := config.DefaultConfig()
	cfg.DefaultPass = "hunter2"
	if err := config.SaveTo(cfg, path); err != nil {
		t.Fatal(err)
	}

	old := cfgFile
	cfgFile = path
	t.Cleanup(func() { cfgFile = old })

	var out bytes.Buffer
	u := ui.NewWithOptions(ui.Options{
		Out:         &out,
		Err:         &bytes.Buffer{},
		Interactive: boolPointer(false),
		Env:         []string{"TERM=dumb"},
	})
	if err := showConfig(u); err != nil {
		t.Fatalf("showConfig: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "********") {
		t.Fatalf("masked password missing from output:\n%s", got)
	}
	if strings.Contains(got, "hunter2") {
		t.Fatalf("password leaked in output:\n%s", got)
	}
	if !strings.Contains(got, path) {
		t.Fatalf("config path missing from output:\n%s", got)
	}
}

func TestEffectiveConfigPathExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := cfgFile
	cfgFile = "~/custom.conf"
	t.Cleanup(func() { cfgFile = old })

	if got, want := effectiveConfigPath(), filepath.Join(home, "custom.conf"); got != want {
		t.Fatalf("effectiveConfigPath = %q, want %q", got, want)
	}
}
