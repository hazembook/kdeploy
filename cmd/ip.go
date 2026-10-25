package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var getIPCmd = &cobra.Command{
	Use:   "ip <vm_name>",
	Short: "Show IP address of a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useSudo := hypervisor.NeedsSudo()
		ip, err := hypervisor.GetDomifaddr(name, useSudo)
		if err != nil {
			return fmt.Errorf("ip lookup: %w", err)
		}
		u := ui.New()
		u.Println(ip)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getIPCmd)
}
