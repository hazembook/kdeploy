package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var shutdownCmd = &cobra.Command{
	Use:   "shutdown <vm_name>",
	Short: "Gracefully shut down a VM (ACPI)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useSudo := hypervisor.NeedsSudo()
		fmt.Printf("⏹  Shutting down VM: %s\n", name)
		if err := hypervisor.VMShutdown(name, useSudo); err != nil {
			return err
		}
		fmt.Printf("✅ VM '%s' shutting down\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(shutdownCmd)
}
