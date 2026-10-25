package ui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/huh/v2"
)

func TestNewPlainMode(t *testing.T) {
	u := NewWithOptions(Options{
		Out:         &bytes.Buffer{},
		Err:         &bytes.Buffer{},
		Interactive: boolPtr(false),
		Env:         []string{"TERM=dumb"},
	})
	if !u.Plain {
		t.Fatal("plain mode should be enabled for a non-interactive terminal")
	}
	if !u.NoColor {
		t.Fatal("plain mode should disable color")
	}
	if u.Width != 80 {
		t.Fatalf("width = %d, want 80", u.Width)
	}
	if u.Symbols.Success != "ok" {
		t.Fatalf("plain success symbol = %q, want ok", u.Symbols.Success)
	}
}

func TestTablePlainNoANSI(t *testing.T) {
	var out bytes.Buffer
	u := NewWithOptions(Options{
		Out:         &out,
		Err:         &bytes.Buffer{},
		Interactive: boolPtr(false),
		Env:         []string{"TERM=dumb"},
	})
	u.Table([]string{"Name", "State"}, [][]string{{"demo", "running"}})

	got := out.String()
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("plain table contains ANSI: %q", got)
	}
	for _, want := range []string{"Name", "State", "demo", "running"} {
		if !strings.Contains(got, want) {
			t.Fatalf("plain table %q does not contain %q", got, want)
		}
	}
}

func TestTablePlainGolden(t *testing.T) {
	var out bytes.Buffer
	u := NewWithOptions(Options{
		Out:         &out,
		Err:         &bytes.Buffer{},
		Interactive: boolPtr(false),
		Env:         []string{"TERM=dumb"},
	})
	u.Table([]string{"Name", "State"}, [][]string{{"demo", "running"}})

	want, err := os.ReadFile(filepath.Join("testdata", "table.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != string(want) {
		t.Fatalf("plain table mismatch:\n got: %q\nwant: %q", got, string(want))
	}
}

func TestKV(t *testing.T) {
	var out bytes.Buffer
	u := NewWithOptions(Options{
		Out:         &out,
		Err:         &bytes.Buffer{},
		Interactive: boolPtr(false),
		Env:         []string{"TERM=dumb"},
	})
	u.KV("Name", "demo", "State", "running")

	got := out.String()
	if !strings.Contains(got, "Name:") || !strings.Contains(got, "demo") {
		t.Fatalf("unexpected KV output: %q", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("plain KV contains ANSI: %q", got)
	}
}

func TestMaskSecret(t *testing.T) {
	u := NewWithOptions(Options{Interactive: boolPtr(false)})
	if got := u.MaskSecret(""); got != "" {
		t.Fatalf("empty secret masked as %q", got)
	}
	if got := u.MaskSecret("hunter2"); got != "********" {
		t.Fatalf("secret masked as %q", got)
	}
}

func TestRunFormRequiresTTY(t *testing.T) {
	u := NewWithOptions(Options{
		Out:         &bytes.Buffer{},
		Err:         &bytes.Buffer{},
		Interactive: boolPtr(false),
	})
	form := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Name")))
	err := u.RunForm(context.Background(), form)
	if !errors.Is(err, ErrNonInteractive) {
		t.Fatalf("RunForm error = %v, want ErrNonInteractive", err)
	}
}

func TestValidators(t *testing.T) {
	if err := NotEmpty("Name")(" "); err == nil {
		t.Fatal("blank value accepted")
	}
	if err := PositiveInt("RAM")("0"); err == nil {
		t.Fatal("zero accepted")
	}
	if err := PositiveInt("RAM")("2048"); err != nil {
		t.Fatalf("valid integer rejected: %v", err)
	}
	if err := HTTPURL("URL")("ftp://example.com"); err == nil {
		t.Fatal("non-http URL accepted")
	}
	if err := HTTPURL("URL")("https://example.com/image.qcow2"); err != nil {
		t.Fatalf("valid URL rejected: %v", err)
	}
	if err := NoWhitespace("Name")("bad name"); err == nil {
		t.Fatal("whitespace accepted")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
