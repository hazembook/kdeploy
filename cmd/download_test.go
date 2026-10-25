package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

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
		"not a url",
	} {
		if _, err := imageFilenameFromURL(rawURL); err == nil {
			t.Errorf("imageFilenameFromURL(%q) accepted", rawURL)
		}
	}
}

func TestRemoveDownloadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.qcow2")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeDownloadFile(path, false); err != nil {
		t.Fatalf("removeDownloadFile: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if err := removeDownloadFile(path, false); err != nil {
		t.Fatalf("removeDownloadFile missing: %v", err)
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
