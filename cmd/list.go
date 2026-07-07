package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all VMs",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		useSudo := hypervisor.NeedsSudo()
		entries, err := hypervisor.List(useSudo)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No VMs found.")
			return nil
		}
		fmt.Printf("%-4s %-20s %s\n", "ID", "Name", "State")
		fmt.Println("-----------------------------")
		for _, e := range entries {
			fmt.Printf("%-4s %-20s %s\n", e.ID, e.Name, e.State)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
