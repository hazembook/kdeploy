package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var consoleCmd = &cobra.Command{
	Use:   "console <vm_name>",
	Short: "Open a serial console to a VM",
	Long: `Opens an interactive serial console to the VM.
Exit with Ctrl+] (Ctrl+]) or ~.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		useSudo := hypervisor.NeedsSudo()
		fmt.Printf("🖥  Connecting to console: %s\n", name)
		return hypervisor.VMConsole(name, useSudo)
	},
}

func init() {
	rootCmd.AddCommand(consoleCmd)
}
