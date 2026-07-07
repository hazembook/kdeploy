package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy <vm_name>",
	Short: "Destroy a VM and clean up resources",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("🔥 Destroying VM: %s\n", name)
		// TODO: wire up virsh destroy/undefine, file cleanup, SSH config removal
		return nil
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
