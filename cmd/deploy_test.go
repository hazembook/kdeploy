package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "images", "nested")
	if err := ensureDir(dir); err != nil {
		t.Fatalf("ensureDir: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("directory not created: %v", err)
	}
	if err := ensureDir(""); err == nil {
		t.Fatal("ensureDir accepted an empty path")
	}
}

func TestEnsureDirRejectsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureDir(path); err == nil {
		t.Fatal("ensureDir accepted a regular file")
	}
}

func TestNetName(t *testing.T) {
	if got := netName("default,mac=52:54:00:12:34:56"); got != "default" {
		t.Fatalf("netName = %q, want default", got)
	}
}
