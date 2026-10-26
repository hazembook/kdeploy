package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <vm_name>",
	Short: "Edit VM XML configuration ($EDITOR)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		u := ui.New()
		u.Status("Editing XML for VM %q", name)
		if err := hypervisor.VMEdit(name, hypervisor.NeedsSudo()); err != nil {
			return fmt.Errorf("edit %s: %w", name, err)
		}
		u.Success("VM %q XML saved", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
