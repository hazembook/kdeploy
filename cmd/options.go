package cmd

import (
	"fmt"
	"strings"

	"github.com/hazembook/kdeploy/pkg/config"
)

const (
	builtinDiskSize = "20G"
	builtinRAM      = "2048"
	builtinCPUs     = "2"
	builtinNetwork  = "default"
	builtinPassword = "linux"
)

// DeployFlagValues contains the deploy-relevant persistent flags. Empty values
// mean the flag was not supplied; config and built-in defaults fill the gap.
type DeployFlagValues struct {
	Size        string
	RAM         string
	CPUs        string
	Network     string
	ImagePath   string
	StoragePath string
}

// DeployOptions is the fully resolved deploy input. It is intentionally free
// of Cobra and terminal state so precedence can be tested directly.
type DeployOptions struct {
	Name        string
	ImagePath   string
	StoragePath string
	Size        string
	RAM         string
	CPUs        string
	Network     string
	Password    string
	Packages    []string
}

// DeployOptionsFromConfig applies config values over the built-in defaults.
func DeployOptionsFromConfig(cfg *config.Config) DeployOptions {
	defaults := config.DefaultConfig()
	if cfg == nil {
		cfg = defaults
	}

	return DeployOptions{
		ImagePath:   firstNonEmpty(cfg.ImagePath, defaults.ImagePath),
		StoragePath: firstNonEmpty(cfg.StoragePath, defaults.StoragePath),
		Size:        firstNonEmpty(cfg.DefaultSize, builtinDiskSize),
		RAM:         firstNonEmpty(cfg.DefaultRAM, builtinRAM),
		CPUs:        firstNonEmpty(cfg.DefaultCPUs, builtinCPUs),
		Network:     builtinNetwork,
		Password:    firstNonEmpty(cfg.DefaultPass, builtinPassword),
	}
}

// ResolveDeployOptions applies flag > config > built-in precedence and then
// validates the result. The optional second positional argument is disk size.
func ResolveDeployOptions(args []string, cfg *config.Config, flags DeployFlagValues) (DeployOptions, error) {
	if len(args) == 0 {
		return DeployOptions{}, fmt.Errorf("VM name is required")
	}
	if len(args) > 2 {
		return DeployOptions{}, fmt.Errorf("expected one or two arguments, got %d", len(args))
	}

	opts := DeployOptionsFromConfig(cfg)
	opts.Name = strings.TrimSpace(args[0])
	opts.Size = firstNonEmpty(flags.Size, opts.Size)
	opts.RAM = firstNonEmpty(flags.RAM, opts.RAM)
	opts.CPUs = firstNonEmpty(flags.CPUs, opts.CPUs)
	opts.Network = firstNonEmpty(flags.Network, opts.Network)
	opts.ImagePath = firstNonEmpty(flags.ImagePath, opts.ImagePath)
	opts.StoragePath = firstNonEmpty(flags.StoragePath, opts.StoragePath)
	if len(args) == 2 {
		opts.Size = strings.TrimSpace(args[1])
	}

	opts.ImagePath = config.ExpandPath(opts.ImagePath)
	opts.StoragePath = config.ExpandPath(opts.StoragePath)
	opts.Size = strings.TrimSpace(opts.Size)
	opts.RAM = strings.TrimSpace(opts.RAM)
	opts.CPUs = strings.TrimSpace(opts.CPUs)
	opts.Network = strings.TrimSpace(opts.Network)

	if err := ValidateVMName(opts.Name); err != nil {
		return DeployOptions{}, err
	}
	if err := ValidateDiskSize(opts.Size); err != nil {
		return DeployOptions{}, err
	}
	if err := ValidateRAM(opts.RAM); err != nil {
		return DeployOptions{}, err
	}
	if err := ValidateCPUs(opts.CPUs); err != nil {
		return DeployOptions{}, err
	}
	if err := ValidateNetwork(opts.Network); err != nil {
		return DeployOptions{}, err
	}
	if err := ValidatePackages(opts.Packages); err != nil {
		return DeployOptions{}, err
	}
	return opts, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
