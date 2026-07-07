package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:   "reboot <vm_name>",
	Short: "Reboot a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useSudo := hypervisor.NeedsSudo()
		fmt.Printf("🔄 Rebooting VM: %s\n", name)
		if err := hypervisor.VMReboot(name, useSudo); err != nil {
			return err
		}
		fmt.Printf("✅ VM '%s' rebooting\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rebootCmd)
}
