package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [vm_name]",
	Short: "Show detailed info for all VMs or a specific VM",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		useSudo := hypervisor.NeedsSudo()

		if len(args) == 1 {
			return showVMInfo(args[0], useSudo)
		}
		return showAllInfo(useSudo)
	},
}

func showVMInfo(name string, useSudo bool) error {
	ip, err := hypervisor.GetDomifaddr(name, useSudo)
	if err != nil {
		ip = "-"
	}

	entries, err := hypervisor.List(useSudo)
	if err != nil {
		return err
	}
	state := "-"
	for _, e := range entries {
		if e.Name == name {
			state = e.State
			break
		}
	}

	fmt.Printf("Name:   %s\nState:  %s\nIP:     %s\n", name, state, ip)
	return nil
}

func showAllInfo(useSudo bool) error {
	entries, err := hypervisor.List(useSudo)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No VMs found.")
		return nil
	}

	fmt.Printf("%-20s %-12s %s\n", "Name", "State", "IP")
	fmt.Println("------------------------------------------------")
	for _, e := range entries {
		ip := "-"
		if e.State == "running" {
			if v, err := hypervisor.GetDomifaddr(e.Name, useSudo); err == nil {
				ip = v
			}
		}
		fmt.Printf("%-20s %-12s %s\n", e.Name, e.State, ip)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
