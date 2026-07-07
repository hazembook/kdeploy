package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckSpace(t *testing.T) {
	dir := t.TempDir()
	info, err := CheckSpace(dir)
	if err != nil {
		t.Fatalf("CheckSpace: %v", err)
	}
	if info.AvailableKB == 0 {
		t.Error("expected non-zero available space")
	}
	if info.TotalKB == 0 {
		t.Error("expected non-zero total space")
	}
}

func TestCheckSpace_File(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	os.WriteFile(f, []byte("data"), 0644)
	_, err := CheckSpace(f)
	if err != nil {
		t.Fatalf("CheckSpace on file: %v", err)
	}
}
