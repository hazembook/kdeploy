package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [vm_name]",
	Short: "Show detailed info for all VMs or a specific VM",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		useSudo := hypervisor.NeedsSudo()

		if len(args) == 1 {
			return showVMInfo(u, args[0], useSudo)
		}
		return showAllInfo(u, useSudo)
	},
}

func showVMInfo(u *ui.UI, name string, useSudo bool) error {
	entries, err := hypervisor.List(useSudo)
	if err != nil {
		return err
	}

	state := ""
	for _, e := range entries {
		if e.Name == name {
			state = e.State
			break
		}
	}
	if state == "" {
		return fmt.Errorf("VM %q not found", name)
	}

	ip := "-"
	if state == "running" {
		if value, err := hypervisor.GetDomifaddr(name, useSudo); err == nil {
			ip = value
		}
	}

	u.KV("Name", name, "State", state, "IP", ip)
	return nil
}

func showAllInfo(u *ui.UI, useSudo bool) error {
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
		ip := "-"
		if e.State == "running" {
			if value, err := hypervisor.GetDomifaddr(e.Name, useSudo); err == nil {
				ip = value
			}
		}
		rows = append(rows, []string{e.Name, e.State, ip})
	}
	u.Table([]string{"Name", "State", "IP"}, rows)
	return nil
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
