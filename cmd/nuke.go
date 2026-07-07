package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var nukeCmd = &cobra.Command{
	Use:   "nuke <vm_name>",
	Short: "Destroy VM and remove all storage",
	Long: `Forcefully destroys and undefines a VM, removing all associated
storage volumes. More aggressive than 'destroy'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useSudo := hypervisor.NeedsSudo()
		fmt.Printf("💥 Nuking VM: %s\n", name)
		if err := hypervisor.VMNuke(name, useSudo); err != nil {
			return err
		}
		fmt.Printf("✅ VM '%s' nuked\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(nukeCmd)
}
