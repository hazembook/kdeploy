package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ImagePath == "" {
		t.Fatal("expected non-empty ImagePath")
	}
	if cfg.StoragePath == "" {
		t.Fatal("expected non-empty StoragePath")
	}
	if cfg.DefaultPass != "linux" {
		t.Fatalf("expected default pass 'linux', got %q", cfg.DefaultPass)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kdeploy.conf")

	cfg := &Config{
		ImagePath:   "/custom/images",
		StoragePath: "/custom/storage",
		DefaultSize: "50G",
		DefaultRAM:  "4096",
		DefaultCPUs: "4",
		DefaultPass: "mypass",
	}

	content := fmt.Sprintf(`# kdeploy configuration
IMAGE_PATH=%s
STORAGE_PATH=%s
DEFAULT_VM_SIZE=%s
DEFAULT_RAM=%s
DEFAULT_CPUS=%s
DEFAULT_PASSWORD=%s
`, cfg.ImagePath, cfg.StoragePath, cfg.DefaultSize,
		cfg.DefaultRAM, cfg.DefaultCPUs, cfg.DefaultPass)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ImagePath != cfg.ImagePath {
		t.Errorf("ImagePath: got %q, want %q", loaded.ImagePath, cfg.ImagePath)
	}
	if loaded.StoragePath != cfg.StoragePath {
		t.Errorf("StoragePath: got %q, want %q", loaded.StoragePath, cfg.StoragePath)
	}
	if loaded.DefaultSize != cfg.DefaultSize {
		t.Errorf("DefaultSize: got %q, want %q", loaded.DefaultSize, cfg.DefaultSize)
	}
	if loaded.DefaultRAM != cfg.DefaultRAM {
		t.Errorf("DefaultRAM: got %q, want %q", loaded.DefaultRAM, cfg.DefaultRAM)
	}
	if loaded.DefaultCPUs != cfg.DefaultCPUs {
		t.Errorf("DefaultCPUs: got %q, want %q", loaded.DefaultCPUs, cfg.DefaultCPUs)
	}
	if loaded.DefaultPass != cfg.DefaultPass {
		t.Errorf("DefaultPass: got %q, want %q", loaded.DefaultPass, cfg.DefaultPass)
	}
}

func TestLoad_NotExist(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config")
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default config when file not found")
	}
}

func TestLoad_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kdeploy.conf")
	os.WriteFile(path, []byte("not=key=value\nBADLINE\n"), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load should handle malformed lines: %v", err)
	}
	if cfg.DefaultPass != "linux" {
		t.Fatalf("expected default pass, got %q", cfg.DefaultPass)
	}
}
