package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/detect"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

const diskSpaceMinKB = 20 * 1024 * 1024 // 20GB

func netName(network string) string {
	return strings.Split(network, ",")[0]
}

var deployCmd = &cobra.Command{
	Use:   "deploy <vm_name> [disk_size]",
	Short: "Deploy a new VM",
	Long: `Creates and boots a new KVM virtual machine from a cloud image.
VM name is required. Optional disk size (default: 20G).`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		if !u.CanPrompt() && !skipConfirm {
			return fmt.Errorf("%w: deploy needs prompts; use --yes to accept defaults", ui.ErrNonInteractive)
		}

		cfg, err := loadDeployConfig(cmd.Context(), u, effectiveConfigPath(), skipConfirm)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}

		opts, err := ResolveDeployOptions(args, cfg, DeployFlagValues{
			Size:        vmSize,
			RAM:         vmRAM,
			CPUs:        vmCPUs,
			Network:     vmNetwork,
			ImagePath:   imagePath,
			StoragePath: storagePath,
		})
		if err != nil {
			return err
		}

		if err := ensureDir(opts.ImagePath); err != nil {
			return fmt.Errorf("image library: %w", err)
		}
		if err := ensureDir(opts.StoragePath); err != nil {
			return fmt.Errorf("VM storage: %w", err)
		}

		keyPath, cmdSudo, err := deployPreflight(u, opts)
		if err != nil {
			return err
		}

		existing := hypervisor.VMExists(opts.Name, cmdSudo)
		if existing && skipConfirm {
			u.Warn("VM %q already exists; it will be destroyed and recreated.", opts.Name)
		}

		var selected detect.ScannedImage
		if skipConfirm {
			selected, err = firstImage(opts.ImagePath)
		} else {
			selected, err = selectImage(cmd.Context(), u, opts.ImagePath)
		}
		if err != nil {
			return err
		}

		if !skipConfirm {
			opts, err = promptResources(cmd.Context(), u, opts)
			if err != nil {
				return err
			}
			opts, selected, err = reviewDeploy(cmd.Context(), u, opts, selected, existing)
			if err != nil {
				return err
			}
		}

		return executeDeploy(cmd.Context(), u, opts, selected, keyPath)
	},
}

func ensureDir(path string) error {
	if path == "" {
		return fmt.Errorf("directory path is empty")
	}
	if err := os.MkdirAll(path, 0755); err == nil {
		return nil
	} else if !hypervisor.NeedsPrivilegedPath(path) {
		return fmt.Errorf("create %s: %w", path, err)
	}

	out, err := exec.Command("sudo", "mkdir", "-p", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("create %s: %w: %s", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVarP(&vmSize, "size", "d", "", "Disk size (e.g. 20G, 50G; default: config or 20G)")
}
