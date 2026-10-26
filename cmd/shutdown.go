package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var shutdownCmd = &cobra.Command{
	Use:   "shutdown <vm_name>",
	Short: "Gracefully shut down a VM (ACPI)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		u := ui.New()
		u.Status("Shutting down VM %q", name)
		if err := hypervisor.VMShutdown(name, hypervisor.NeedsSudo()); err != nil {
			return fmt.Errorf("shutdown %s: %w", name, err)
		}
		u.Success("VM %q shutting down", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(shutdownCmd)
}
