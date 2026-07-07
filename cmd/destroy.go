package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hazembook/kdeploy/pkg/config"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/hazembook/kdeploy/pkg/ssh"
	"github.com/spf13/cobra"
)

func sudoRemove(path string) error {
	pre := hypervisor.SudoForPath(path)
	if pre == nil {
		return os.Remove(path)
	}
	return exec.Command(pre[0], "rm", "-f", path).Run()
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <vm_name>",
	Short: "Destroy a VM and clean up resources",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("🔥 Destroying VM: %s\n", name)

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		storPath := cfg.StoragePath
		if storagePath != "" {
			storPath = storagePath
		}

		useSudo := hypervisor.NeedsSudo()

		hypervisor.Destroy(name, useSudo)

		sudoRemove(filepath.Join(storPath, name+".qcow2"))
		sudoRemove(filepath.Join(storPath, name+"-seed.iso"))

		incDir, _ := ssh.IncludeDir()
		ssh.RemoveHostConfig(name, incDir)

		fmt.Printf("✅ VM '%s' destroyed and cleaned up\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
