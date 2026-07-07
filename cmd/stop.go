package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <vm_name>",
	Short: "Force-stop a VM (virsh destroy)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useSudo := hypervisor.NeedsSudo()
		fmt.Printf("⏹  Force-stopping VM: %s\n", name)
		if err := hypervisor.VMStop(name, useSudo); err != nil {
			return err
		}
		fmt.Printf("✅ VM '%s' stopped\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
