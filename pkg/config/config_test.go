package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, ".config", "kdeploy.conf")
	if got := DefaultPath(); got != want {
		t.Fatalf("DefaultPath = %q, want %q", got, want)
	}
}

func TestExpandPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tests := []struct {
		in   string
		want string
	}{
		{"~", home},
		{"~/VM/cloud_images", filepath.Join(home, "VM", "cloud_images")},
		{"/var/lib/libvirt/images", "/var/lib/libvirt/images"},
		{"~user/images", "~user/images"},
		{"  ~/spaced  ", filepath.Join(home, "spaced")},
	}
	for _, tt := range tests {
		if got := ExpandPath(tt.in); got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kdeploy.conf")
	if Exists(path) {
		t.Fatal("missing file reported as existing")
	}
	if err := os.WriteFile(path, []byte("IMAGE_PATH=/tmp/images\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if !Exists(path) {
		t.Fatal("existing file reported as missing")
	}
}

func TestSaveToAtomicPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kdeploy.conf")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	cfg.ImagePath = "~/VM/cloud_images"
	cfg.StoragePath = "~/VM/storage"
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "~/") {
		t.Fatalf("SaveTo wrote unexpanded path:\n%s", data)
	}
	temps, err := filepath.Glob(filepath.Join(dir, ".kdeploy-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temps) != 0 {
		t.Fatalf("temporary files left behind: %v", temps)
	}
}

func TestSaveToAndLoadRoundTrip(t *testing.T) {
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
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *loaded != *cfg {
		t.Fatalf("round trip = %+v, want %+v", loaded, cfg)
	}
}

func TestLoadExpandsPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	content := "IMAGE_PATH=~/images\nSTORAGE_PATH=~/storage\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ImagePath != filepath.Join(home, "images") {
		t.Fatalf("ImagePath = %q", loaded.ImagePath)
	}
	if loaded.StoragePath != filepath.Join(home, "storage") {
		t.Fatalf("StoragePath = %q", loaded.StoragePath)
	}
}

func TestSaveToNilConfig(t *testing.T) {
	if err := SaveTo(nil, filepath.Join(t.TempDir(), "kdeploy.conf")); err == nil {
		t.Fatal("SaveTo(nil) succeeded")
	}
}
