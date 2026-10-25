package cmd

import (
	"context"
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/config"
)

const (
	imagePresetHome   = "~/VM/cloud_images"
	imagePresetSystem = "/var/lib/libvirt/images"
	storagePresetHome = "~/VM/disks"
	presetCustom      = "__custom__"
)

// SetupOptions controls the shared first-run / --reconfig / config-init path.
type SetupOptions struct {
	Path        string
	Write       bool
	Force       bool
	Reset       bool
	Interactive bool
}

// runSetup loads the existing config when present, optionally prompts for the
// two path choices, and atomically saves the result. It never stores a
// password in the clear beyond the existing config file contract.
func runSetup(ctx context.Context, u *ui.UI, opts SetupOptions) (*config.Config, error) {
	path := config.ExpandPath(opts.Path)
	if path == "" {
		path = config.DefaultPath()
	}

	cfg := config.DefaultConfig()
	if !opts.Reset && config.Exists(path) {
		loaded, err := config.Load(path)
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		cfg = loaded
	}

	if opts.Write && !opts.Force && config.Exists(path) {
		return nil, fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
	}

	if opts.Interactive && u.CanPrompt() {
		state := setupPathStateFor(cfg)
		imageChoice := state.ImageChoice
		storageChoice := state.StorageChoice
		customImage := state.CustomImage
		customStorage := state.CustomStorage

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Image library").
					Description("Where cloud images are stored. Paths beginning with ~ are expanded to your home directory.").
					Options(
						huh.NewOption("~/VM/cloud_images  (user-owned, recommended)", imagePresetHome),
						huh.NewOption("/var/lib/libvirt/images  (system path, may prompt for sudo)", imagePresetSystem),
						huh.NewOption("Custom path…", presetCustom),
					).
					Value(&imageChoice),
				huh.NewSelect[string]().
					Title("VM storage").
					Description("Where disk overlays and seed ISOs are kept.").
					Options(
						huh.NewOption("/var/lib/libvirt/images  (libvirt standard, may prompt for sudo)", imagePresetSystem),
						huh.NewOption("~/VM/disks  (user-owned)", storagePresetHome),
						huh.NewOption("Custom path…", presetCustom),
					).
					Value(&storageChoice),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("Custom image library path").
					Placeholder("~/VM/cloud_images").
					Value(&customImage).
					Validate(func(value string) error {
						if strings.TrimSpace(value) == "" {
							return fmt.Errorf("image library path is required")
						}
						return nil
					}),
			).WithHideFunc(func() bool { return imageChoice != presetCustom }),
			huh.NewGroup(
				huh.NewInput().
					Title("Custom VM storage path").
					Placeholder("~/VM/disks").
					Value(&customStorage).
					Validate(func(value string) error {
						if strings.TrimSpace(value) == "" {
							return fmt.Errorf("VM storage path is required")
						}
						return nil
					}),
			).WithHideFunc(func() bool { return storageChoice != presetCustom }),
		)

		if err := u.RunForm(ctx, form); err != nil {
			return nil, err
		}

		if imageChoice == presetCustom {
			cfg.ImagePath = config.ExpandPath(customImage)
		} else {
			cfg.ImagePath = config.ExpandPath(imageChoice)
		}
		if storageChoice == presetCustom {
			cfg.StoragePath = config.ExpandPath(customStorage)
		} else {
			cfg.StoragePath = config.ExpandPath(storageChoice)
		}
	}

	cfg.ImagePath = config.ExpandPath(cfg.ImagePath)
	cfg.StoragePath = config.ExpandPath(cfg.StoragePath)
	if opts.Write {
		if err := config.SaveTo(cfg, path); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
		u.Success("Configuration saved to %s", path)
	}
	return cfg, nil
}

// loadDeployConfig loads the configured file, or runs the first-run wizard
// when interactive. A non-interactive deploy (--yes) deliberately returns
// defaults without writing a config file.
func loadDeployConfig(ctx context.Context, u *ui.UI, path string, nonInteractive bool) (*config.Config, error) {
	path = config.ExpandPath(path)
	if path == "" {
		path = config.DefaultPath()
	}
	if config.Exists(path) {
		return config.Load(path)
	}
	if !nonInteractive && u.CanPrompt() {
		u.Section("First-run setup")
		return runSetup(ctx, u, SetupOptions{
			Path:        path,
			Write:       true,
			Interactive: true,
		})
	}
	return config.DefaultConfig(), nil
}

type setupPathState struct {
	ImageChoice   string
	StorageChoice string
	CustomImage   string
	CustomStorage string
}

func setupPathStateFor(cfg *config.Config) setupPathState {
	state := setupPathState{
		ImageChoice:   imagePresetHome,
		StorageChoice: imagePresetSystem,
	}
	if cfg == nil {
		return state
	}

	switch cfg.ImagePath {
	case "", config.ExpandPath(imagePresetHome):
		state.ImageChoice = imagePresetHome
	case imagePresetSystem:
		state.ImageChoice = imagePresetSystem
	default:
		state.ImageChoice = presetCustom
		state.CustomImage = cfg.ImagePath
	}

	switch cfg.StoragePath {
	case "", imagePresetSystem:
		state.StorageChoice = imagePresetSystem
	case config.ExpandPath(storagePresetHome):
		state.StorageChoice = storagePresetHome
	default:
		state.StorageChoice = presetCustom
		state.CustomStorage = cfg.StoragePath
	}
	return state
}
