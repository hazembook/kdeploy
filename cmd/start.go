package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <vm_name>",
	Short: "Start a stopped VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useSudo := hypervisor.NeedsSudo()
		fmt.Printf("▶️  Starting VM: %s\n", name)
		if err := hypervisor.VMStart(name, useSudo); err != nil {
			return err
		}
		fmt.Printf("✅ VM '%s' started\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
