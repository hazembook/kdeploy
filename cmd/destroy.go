package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/config"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/hazembook/kdeploy/pkg/ssh"
	"github.com/spf13/cobra"
)

func sudoRemove(path string) error {
	if err := os.Remove(path); err == nil || os.IsNotExist(err) {
		return nil
	}
	out, err := exec.Command("sudo", "rm", "-f", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove %s: %w: %s", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func vmResourcePaths(storagePath, name string) (disk, seed, sshConfig string, err error) {
	includeDir, err := ssh.IncludeDir()
	if err != nil {
		return "", "", "", err
	}
	return filepath.Join(storagePath, name+".qcow2"),
		filepath.Join(storagePath, name+"-seed.iso"),
		filepath.Join(includeDir, name+".conf"),
		nil
}

func existingResources(paths ...string) ([]string, error) {
	var found []string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			found = append(found, path)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect %s: %w", path, err)
		}
	}
	return found, nil
}

func removeVMResources(name, storagePath string) error {
	disk, seed, sshConfig, err := vmResourcePaths(storagePath, name)
	if err != nil {
		return fmt.Errorf("ssh config path: %w", err)
	}
	for _, path := range []string{disk, seed} {
		if err := sudoRemove(path); err != nil {
			return err
		}
	}
	if err := ssh.RemoveHostConfig(name, filepath.Dir(sshConfig)); err != nil {
		return err
	}
	return nil
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <vm_name>",
	Short: "Destroy a VM and clean up resources",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		if !u.CanPrompt() && !skipConfirm {
			return fmt.Errorf("%w: destroy needs confirmation; use --yes", ui.ErrNonInteractive)
		}

		name := args[0]
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		storPath := config.ExpandPath(cfg.StoragePath)
		if storagePath != "" {
			storPath = config.ExpandPath(storagePath)
		}

		useSudo := hypervisor.NeedsSudo()
		exists := hypervisor.VMExists(name, useSudo)
		disk, seed, sshConfig, err := vmResourcePaths(storPath, name)
		if err != nil {
			return fmt.Errorf("ssh config path: %w", err)
		}

		leftovers, err := existingResources(disk, seed, sshConfig)
		if err != nil {
			return err
		}
		if !exists && len(leftovers) == 0 {
			u.Warn("VM %q does not exist and has no leftover resources.", name)
			u.KV("Name", name, "VM", "not found", "Resources", "none")
			return nil
		}

		definition := "will be undefined"
		if !exists {
			definition = "not present"
		}
		u.Panel("Removal plan",
			"VM", name,
			"Definition", definition,
			"Disk", disk,
			"Seed", seed,
			"SSH config", sshConfig,
		)

		if !skipConfirm {
			confirm := false
			question := fmt.Sprintf("Remove leftover resources for VM %q?", name)
			if exists {
				question = fmt.Sprintf("Destroy VM %q and remove its disk, seed ISO, and SSH config?", name)
			}
			if err := u.Confirm(cmd.Context(), question, &confirm); err != nil {
				return err
			}
			if !confirm {
				return fmt.Errorf("destroy cancelled")
			}
		}

		if exists {
			u.Status("Destroying VM %q", name)
			if err := hypervisor.Destroy(name, useSudo); err != nil {
				return fmt.Errorf("destroy %q: %w", name, err)
			}
			if hypervisor.VMExists(name, useSudo) {
				return fmt.Errorf("VM %q still exists after virsh undefine", name)
			}
		}

		if err := removeVMResources(name, storPath); err != nil {
			return fmt.Errorf("cleanup: %w", err)
		}

		if exists {
			u.Success("VM %q destroyed", name)
			u.KV("Name", name, "VM", "destroyed", "Resources", "cleaned")
		} else {
			u.Success("Leftover resources for %q removed", name)
			u.KV("Name", name, "VM", "not found", "Resources", "cleaned")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
