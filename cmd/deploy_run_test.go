package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPubKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(keyPath+".pub", []byte("ssh-ed25519 AAAA test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := readPubKey(keyPath); got != "ssh-ed25519 AAAA test" {
		t.Fatalf("readPubKey = %q", got)
	}
	if got := readPubKey(filepath.Join(dir, "missing")); got != "" {
		t.Fatalf("missing key = %q", got)
	}
}
