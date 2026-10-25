package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ImagePath   string
	StoragePath string
	DefaultSize string
	DefaultRAM  string
	DefaultCPUs string
	DefaultPass string
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		ImagePath:   filepath.Join(home, "VM", "cloud_images"),
		StoragePath: "/var/lib/libvirt/images",
		DefaultSize: "20G",
		DefaultRAM:  "2048",
		DefaultCPUs: "2",
		DefaultPass: "linux",
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".config", "kdeploy.conf")
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "IMAGE_PATH":
			cfg.ImagePath = val
		case "STORAGE_PATH":
			cfg.StoragePath = val
		case "DEFAULT_VM_SIZE":
			cfg.DefaultSize = val
		case "DEFAULT_RAM":
			cfg.DefaultRAM = val
		case "DEFAULT_CPUS":
			cfg.DefaultCPUs = val
		case "DEFAULT_PASSWORD":
			cfg.DefaultPass = val
		}
	}
	return cfg, scanner.Err()
}

func Save(cfg *Config) error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "kdeploy.conf")

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
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

	return os.WriteFile(path, []byte(content), 0600)
}
