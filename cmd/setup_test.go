package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/config"
)

func setupTestUI() *ui.UI {
	return ui.NewWithOptions(ui.Options{
		Out:         &bytes.Buffer{},
		Err:         &bytes.Buffer{},
		Interactive: boolPointer(false),
		Env:         []string{"TERM=dumb"},
	})
}

func boolPointer(value bool) *bool { return &value }

func TestRunSetupWritesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	cfg, err := runSetup(context.Background(), setupTestUI(), SetupOptions{
		Path:  path,
		Write: true,
	})
	if err != nil {
		t.Fatalf("runSetup: %v", err)
	}
	if cfg == nil {
		t.Fatal("nil config")
	}
	if !config.Exists(path) {
		t.Fatal("config was not written")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("config mode = %o, want 600", got)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultPass != cfg.DefaultPass {
		t.Fatalf("password changed: %q vs %q", loaded.DefaultPass, cfg.DefaultPass)
	}
}

func TestRunSetupPreservesExistingValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	original := &config.Config{
		ImagePath:   "/custom/images",
		StoragePath: "/custom/storage",
		DefaultSize: "80G",
		DefaultRAM:  "8192",
		DefaultCPUs: "8",
		DefaultPass: "keep-me",
	}
	if err := config.SaveTo(original, path); err != nil {
		t.Fatal(err)
	}

	cfg, err := runSetup(context.Background(), setupTestUI(), SetupOptions{
		Path:  path,
		Write: true,
		Force: true,
	})
	if err != nil {
		t.Fatalf("runSetup: %v", err)
	}
	if cfg.DefaultPass != "keep-me" || cfg.DefaultSize != "80G" {
		t.Fatalf("existing values not preserved: %+v", cfg)
	}
}

func TestRunSetupResetUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	original := config.DefaultConfig()
	original.DefaultPass = "old"
	original.DefaultSize = "99G"
	if err := config.SaveTo(original, path); err != nil {
		t.Fatal(err)
	}
	cfg, err := runSetup(context.Background(), setupTestUI(), SetupOptions{
		Path:  path,
		Write: true,
		Force: true,
		Reset: true,
	})
	if err != nil {
		t.Fatalf("runSetup: %v", err)
	}
	if cfg.DefaultPass != "linux" || cfg.DefaultSize != "20G" {
		t.Fatalf("Reset did not restore defaults: %+v", cfg)
	}
}

func TestRunSetupRefusesExistingWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	if err := config.SaveTo(config.DefaultConfig(), path); err != nil {
		t.Fatal(err)
	}
	if _, err := runSetup(context.Background(), setupTestUI(), SetupOptions{
		Path:  path,
		Write: true,
	}); err == nil {
		t.Fatal("runSetup overwrote an existing config without --force")
	}
}

func TestRunSetupNoWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	if _, err := runSetup(context.Background(), setupTestUI(), SetupOptions{Path: path}); err != nil {
		t.Fatalf("runSetup: %v", err)
	}
	if config.Exists(path) {
		t.Fatal("runSetup wrote a config when Write was false")
	}
}

func TestRunSetupExpandsConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg, err := runSetup(context.Background(), setupTestUI(), SetupOptions{
		Path:  "~/custom.conf",
		Write: false,
	})
	if err != nil {
		t.Fatalf("runSetup: %v", err)
	}
	if cfg == nil {
		t.Fatal("nil config")
	}
	if config.Exists(filepath.Join(home, "custom.conf")) {
		t.Fatal("Write=false still wrote a file")
	}
}

func TestLoadDeployConfigSkipDoesNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	cfg, err := loadDeployConfig(context.Background(), setupTestUI(), path, true)
	if err != nil {
		t.Fatalf("loadDeployConfig: %v", err)
	}
	if cfg == nil || cfg.DefaultSize != builtinDiskSize {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if config.Exists(path) {
		t.Fatal("non-interactive deploy wrote a config file")
	}
}

func TestLoadDeployConfigLoadsExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kdeploy.conf")
	want := config.DefaultConfig()
	want.DefaultSize = "77G"
	want.DefaultPass = "from-file"
	if err := config.SaveTo(want, path); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadDeployConfig(context.Background(), setupTestUI(), path, true)
	if err != nil {
		t.Fatalf("loadDeployConfig: %v", err)
	}
	if cfg.DefaultSize != "77G" || cfg.DefaultPass != "from-file" {
		t.Fatalf("config not loaded: %+v", cfg)
	}
}

func TestSetupPathStateFor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaults := config.DefaultConfig()
	state := setupPathStateFor(defaults)
	if state.ImageChoice != imagePresetHome || state.CustomImage != "" {
		t.Errorf("default image state = %+v", state)
	}
	if state.StorageChoice != imagePresetSystem || state.CustomStorage != "" {
		t.Errorf("default storage state = %+v", state)
	}

	custom := &config.Config{
		ImagePath:   "/custom/images",
		StoragePath: "/custom/storage",
	}
	state = setupPathStateFor(custom)
	if state.ImageChoice != presetCustom || state.CustomImage != "/custom/images" {
		t.Errorf("custom image state = %+v", state)
	}
	if state.StorageChoice != presetCustom || state.CustomStorage != "/custom/storage" {
		t.Errorf("custom storage state = %+v", state)
	}

	preset := &config.Config{
		ImagePath:   filepath.Join(home, "VM", "cloud_images"),
		StoragePath: filepath.Join(home, "VM", "disks"),
	}
	state = setupPathStateFor(preset)
	if state.ImageChoice != imagePresetHome || state.StorageChoice != storagePresetHome {
		t.Errorf("preset state = %+v", state)
	}
	if state.CustomImage != "" || state.CustomStorage != "" {
		t.Errorf("preset custom fields should be blank: %+v", state)
	}
}
