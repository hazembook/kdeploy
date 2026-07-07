package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <vm_name>",
	Short: "Edit VM XML configuration ($EDITOR)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useSudo := hypervisor.NeedsSudo()
		fmt.Printf("✏️  Editing XML config for: %s\n", name)
		return hypervisor.VMEdit(name, useSudo)
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
