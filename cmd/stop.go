package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <vm_name>",
	Short: "Force-stop a VM (virsh destroy)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		u := ui.New()
		u.Status("Force-stopping VM %q", name)
		if err := hypervisor.VMStop(name, hypervisor.NeedsSudo()); err != nil {
			return fmt.Errorf("stop %s: %w", name, err)
		}
		u.Success("VM %q stopped", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
