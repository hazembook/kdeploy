package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy <vm_name> [disk_size]",
	Short: "Deploy a new VM",
	Long: `Creates and boots a new KVM virtual machine from a cloud image.
VM name is required. Optional disk size (default: 20G).`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		size := vmSize
		if len(args) > 1 {
			size = args[1]
		}
		fmt.Printf("🚀 Deploying VM: %s (disk=%s, ram=%s, cpus=%s, net=%s)\n",
			name, size, vmRAM, vmCPUs, vmNetwork)
		// TODO: wire up config loading, cloud-init gen, virt-install, IP discovery, SSH wait
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVarP(&vmSize, "size", "d", "20G", "Disk size (e.g. 20G, 50G)")
}
