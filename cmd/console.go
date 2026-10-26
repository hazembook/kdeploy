package cmd

import (
	"github.com/hazembook/kdeploy/internal/ui"
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
		u := ui.New()
		u.Status("Opening console for VM %q", name)
		return hypervisor.VMConsole(name, hypervisor.NeedsSudo())
	},
}

func init() {
	rootCmd.AddCommand(consoleCmd)
}
