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

// DefaultPath returns the historical config location:
// ~/.config/kdeploy.conf.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".config", "kdeploy.conf")
	}
	return filepath.Join(home, ".config", "kdeploy.conf")
}

// ExpandPath expands a leading ~ or ~/ using the current user's home.
func ExpandPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}

// Exists reports whether path exists. An empty path means the default.
func Exists(path string) bool {
	path = ExpandPath(path)
	if path == "" {
		path = DefaultPath()
	}
	_, err := os.Stat(path)
	return err == nil
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	path = ExpandPath(path)
	if path == "" {
		path = DefaultPath()
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
			cfg.ImagePath = ExpandPath(val)
		case "STORAGE_PATH":
			cfg.StoragePath = ExpandPath(val)
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
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return cfg, nil
}

func Save(cfg *Config) error {
	return SaveTo(cfg, DefaultPath())
}

// SaveTo atomically writes a config file with mode 0600.
func SaveTo(cfg *Config, path string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	path = ExpandPath(path)
	if path == "" {
		path = DefaultPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}

	out := *cfg
	out.ImagePath = ExpandPath(out.ImagePath)
	out.StoragePath = ExpandPath(out.StoragePath)
	content := fmt.Sprintf(`# kdeploy configuration
IMAGE_PATH=%s
STORAGE_PATH=%s
DEFAULT_VM_SIZE=%s
DEFAULT_RAM=%s
DEFAULT_CPUS=%s
DEFAULT_PASSWORD=%s
`, out.ImagePath, out.StoragePath, out.DefaultSize,
		out.DefaultRAM, out.DefaultCPUs, out.DefaultPass)

	tmp, err := os.CreateTemp(dir, ".kdeploy-*.tmp")
	if err != nil {
		return fmt.Errorf("create config temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod config temp: %w", err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write config temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close config temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}
