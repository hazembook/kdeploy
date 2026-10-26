package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:   "reboot <vm_name>",
	Short: "Reboot a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		u := ui.New()
		u.Status("Rebooting VM %q", name)
		if err := hypervisor.VMReboot(name, hypervisor.NeedsSudo()); err != nil {
			return fmt.Errorf("reboot %s: %w", name, err)
		}
		u.Success("VM %q rebooting", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rebootCmd)
}
