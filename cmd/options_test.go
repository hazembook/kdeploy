package cmd

import (
	"path/filepath"
	"testing"

	"github.com/hazembook/kdeploy/pkg/config"
)

func TestResolveDeployOptionsDefaults(t *testing.T) {
	opts, err := ResolveDeployOptions([]string{"demo"}, nil, DeployFlagValues{})
	if err != nil {
		t.Fatalf("ResolveDeployOptions: %v", err)
	}
	if opts.Name != "demo" {
		t.Errorf("Name = %q", opts.Name)
	}
	if opts.Size != builtinDiskSize {
		t.Errorf("Size = %q, want %q", opts.Size, builtinDiskSize)
	}
	if opts.RAM != builtinRAM {
		t.Errorf("RAM = %q, want %q", opts.RAM, builtinRAM)
	}
	if opts.CPUs != builtinCPUs {
		t.Errorf("CPUs = %q, want %q", opts.CPUs, builtinCPUs)
	}
	if opts.Network != builtinNetwork {
		t.Errorf("Network = %q, want %q", opts.Network, builtinNetwork)
	}
	if opts.Password != builtinPassword {
		t.Errorf("Password = %q, want %q", opts.Password, builtinPassword)
	}
	if opts.ImagePath == "" || opts.StoragePath == "" {
		t.Errorf("paths not populated: %+v", opts)
	}
}

func TestResolveDeployOptionsConfigOverridesDefaults(t *testing.T) {
	cfg := &config.Config{
		ImagePath:   "/images",
		StoragePath: "/storage",
		DefaultSize: "40G",
		DefaultRAM:  "4096",
		DefaultCPUs: "4",
		DefaultPass: "secret",
	}
	opts, err := ResolveDeployOptions([]string{"demo"}, cfg, DeployFlagValues{})
	if err != nil {
		t.Fatalf("ResolveDeployOptions: %v", err)
	}
	if opts.Size != "40G" || opts.RAM != "4096" || opts.CPUs != "4" {
		t.Errorf("config defaults not applied: %+v", opts)
	}
	if opts.ImagePath != "/images" || opts.StoragePath != "/storage" {
		t.Errorf("config paths not applied: %+v", opts)
	}
	if opts.Password != "secret" {
		t.Errorf("Password = %q", opts.Password)
	}
}

func TestResolveDeployOptionsFlagsAndPositionalWin(t *testing.T) {
	cfg := &config.Config{
		ImagePath:   "/config/images",
		StoragePath: "/config/storage",
		DefaultSize: "40G",
		DefaultRAM:  "4096",
		DefaultCPUs: "4",
		DefaultPass: "secret",
	}
	opts, err := ResolveDeployOptions([]string{"demo", "50G"}, cfg, DeployFlagValues{
		Size:        "30G",
		RAM:         "8192",
		CPUs:        "8",
		Network:     "br0",
		ImagePath:   "/flag/images",
		StoragePath: "/flag/storage",
	})
	if err != nil {
		t.Fatalf("ResolveDeployOptions: %v", err)
	}
	if opts.Size != "50G" {
		t.Errorf("positional Size = %q, want 50G", opts.Size)
	}
	if opts.RAM != "8192" || opts.CPUs != "8" || opts.Network != "br0" {
		t.Errorf("flags not applied: %+v", opts)
	}
	if opts.ImagePath != "/flag/images" || opts.StoragePath != "/flag/storage" {
		t.Errorf("path flags not applied: %+v", opts)
	}
}

func TestResolveDeployOptionsSizeFlagBeatsConfigWithoutPositional(t *testing.T) {
	cfg := &config.Config{DefaultSize: "40G", DefaultRAM: builtinRAM, DefaultCPUs: builtinCPUs}
	opts, err := ResolveDeployOptions([]string{"demo"}, cfg, DeployFlagValues{Size: "30G"})
	if err != nil {
		t.Fatalf("ResolveDeployOptions: %v", err)
	}
	if opts.Size != "30G" {
		t.Errorf("Size = %q, want 30G", opts.Size)
	}
}

func TestResolveDeployOptionsExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := &config.Config{ImagePath: "~/images", StoragePath: "~/storage", DefaultPass: builtinPassword}
	opts, err := ResolveDeployOptions([]string{"demo"}, cfg, DeployFlagValues{})
	if err != nil {
		t.Fatalf("ResolveDeployOptions: %v", err)
	}
	if opts.ImagePath != filepath.Join(home, "images") {
		t.Errorf("ImagePath = %q", opts.ImagePath)
	}
	if opts.StoragePath != filepath.Join(home, "storage") {
		t.Errorf("StoragePath = %q", opts.StoragePath)
	}
}

func TestResolveDeployOptionsRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		cfg  *config.Config
		flag DeployFlagValues
	}{
		{"missing name", nil, nil, DeployFlagValues{}},
		{"extra args", []string{"demo", "20G", "extra"}, nil, DeployFlagValues{}},
		{"bad name", []string{"-bad"}, nil, DeployFlagValues{}},
		{"bad size", []string{"demo", "20"}, nil, DeployFlagValues{}},
		{"bad ram", []string{"demo"}, nil, DeployFlagValues{RAM: "0"}},
		{"bad cpus", []string{"demo"}, nil, DeployFlagValues{CPUs: "x"}},
		{"bad network", []string{"demo"}, nil, DeployFlagValues{Network: "bad network"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ResolveDeployOptions(tt.args, tt.cfg, tt.flag); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
