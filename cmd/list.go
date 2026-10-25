package cmd

import (
	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all VMs",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		useSudo := hypervisor.NeedsSudo()
		entries, err := hypervisor.List(useSudo)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			u.Println("No VMs found.")
			return nil
		}

		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, []string{e.ID, e.Name, e.State})
		}
		u.Table([]string{"ID", "Name", "State"}, rows)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
