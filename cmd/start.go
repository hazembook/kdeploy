package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <vm_name>",
	Short: "Start a stopped VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		u := ui.New()
		u.Status("Starting VM %q", name)
		if err := hypervisor.VMStart(name, hypervisor.NeedsSudo()); err != nil {
			return fmt.Errorf("start %s: %w", name, err)
		}
		u.Success("VM %q started", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
