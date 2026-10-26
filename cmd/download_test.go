package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hazembook/kdeploy/internal/ui"
)

func boolPtr(v bool) *bool { return &v }

func TestImageFilenameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/images/Rocky-10.qcow2", "Rocky-10.qcow2"},
		{"http://example.com/a/b/Ubuntu.img?download=1", "Ubuntu.img"},
		{"https://example.com/images/with%20space.qcow2", "with space.qcow2"},
	}
	for _, tt := range tests {
		got, err := imageFilenameFromURL(tt.url)
		if err != nil {
			t.Errorf("imageFilenameFromURL(%q): %v", tt.url, err)
			continue
		}
		if got != tt.want {
			t.Errorf("imageFilenameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}

	for _, rawURL := range []string{
		"",
		"ftp://example.com/image.qcow2",
		"https://example.com/",
		"https://example.com",
		"https://example.com/images/%2e%2e%2Fevil.qcow2",
		"https://example.com/images/..",
		"not a url",
	} {
		if _, err := imageFilenameFromURL(rawURL); err == nil {
			t.Errorf("imageFilenameFromURL(%q) accepted", rawURL)
		}
	}
}

func TestResolveDownloadArg(t *testing.T) {
	catalog := buildCatalog()
	url, err := resolveDownloadArg(catalog, catalog[0].Key)
	if err != nil {
		t.Fatalf("resolve known key: %v", err)
	}
	if url != catalog[0].URL {
		t.Fatalf("known key URL = %q, want %q", url, catalog[0].URL)
	}

	custom := "https://example.com/custom.qcow2"
	url, err = resolveDownloadArg(catalog, custom)
	if err != nil || url != custom {
		t.Fatalf("resolve URL = %q, %v", url, err)
	}

	if _, err := resolveDownloadArg(catalog, "missing"); err == nil {
		t.Fatal("unknown key accepted")
	}
}

func TestConfirmDownloadOverwrite(t *testing.T) {
	oldSkip := skipConfirm
	defer func() { skipConfirm = oldSkip }()

	dir := t.TempDir()
	dest := filepath.Join(dir, "image.qcow2")
	if err := os.WriteFile(dest, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	u := ui.NewWithOptions(ui.Options{
		Out:         os.Stdout,
		Err:         os.Stderr,
		Interactive: boolPtr(false),
		Env:         []string{"TERM=dumb"},
	})

	skipConfirm = true
	confirmed, err := confirmDownloadOverwrite(context.Background(), u, dest, "image.qcow2")
	if err != nil || !confirmed {
		t.Fatalf("--yes overwrite = %v, %v", confirmed, err)
	}

	skipConfirm = false
	if _, err := confirmDownloadOverwrite(context.Background(), u, dest, "image.qcow2"); !errors.Is(err, ui.ErrNonInteractive) {
		t.Fatalf("non-interactive overwrite error = %v, want ErrNonInteractive", err)
	}

	missing := filepath.Join(dir, "missing.qcow2")
	confirmed, err = confirmDownloadOverwrite(context.Background(), u, missing, "missing.qcow2")
	if err != nil || confirmed {
		t.Fatalf("missing destination = %v, %v", confirmed, err)
	}
}

func TestBuildCatalog(t *testing.T) {
	entries := buildCatalog()
	if len(entries) == 0 {
		t.Fatal("empty catalog")
	}
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.Key == "" || entry.Name == "" || entry.URL == "" {
			t.Errorf("incomplete entry: %+v", entry)
		}
		if seen[entry.Key] {
			t.Errorf("duplicate catalog key %q", entry.Key)
		}
		seen[entry.Key] = true
	}
}
