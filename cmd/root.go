package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	skipConfirm bool
	vmRAM       string
	vmCPUs      string
	vmSize      string
	vmNetwork   string
	imagePath   string
	storagePath string
)

var rootCmd = &cobra.Command{
	Use:   "kdeploy",
	Short: "KVM Cloud Image Deployer",
	Long: `kdeploy automates the creation of KVM virtual machines
using cloud-init enabled cloud images (qcow2 format).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if reconfig, _ := cmd.Flags().GetBool("reconfig"); reconfig {
			return configInitCmd.RunE(configInitCmd, nil)
		}
		if showCfg, _ := cmd.Flags().GetBool("show-config"); showCfg {
			return configShowCmd.RunE(configShowCmd, nil)
		}
		return cmd.Help()
	},
}

func Execute(version string) {
	configureCommandGroups()
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version),
		fang.WithNotifySignal(os.Interrupt),
		fang.WithoutCompletions(),
		fang.WithoutManpage(),
	); err != nil {
		os.Exit(1)
	}
}

func configureCommandGroups() {
	rootCmd.AddGroup(
		&cobra.Group{ID: "vm", Title: "VM Lifecycle"},
		&cobra.Group{ID: "inspect", Title: "Inspect"},
		&cobra.Group{ID: "images", Title: "Images"},
		&cobra.Group{ID: "config", Title: "Configuration"},
	)

	for _, command := range rootCmd.Commands() {
		switch command.Name() {
		case "deploy", "destroy", "start", "shutdown", "stop", "reboot":
			command.GroupID = "vm"
		case "list", "info", "ip", "console", "edit":
			command.GroupID = "inspect"
		case "download":
			command.GroupID = "images"
		case "config", "check":
			command.GroupID = "config"
		}
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/kdeploy.conf)")
	rootCmd.PersistentFlags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip all confirmations")
	rootCmd.PersistentFlags().StringVarP(&vmRAM, "ram", "r", "2048", "RAM in MB")
	rootCmd.PersistentFlags().StringVarP(&vmCPUs, "cpu", "c", "2", "Number of vCPUs")
	rootCmd.PersistentFlags().StringVarP(&vmNetwork, "network", "n", "default", "Virtual network to use")
	rootCmd.PersistentFlags().StringVarP(&imagePath, "image-path", "i", "", "Override image library path")
	rootCmd.PersistentFlags().StringVarP(&storagePath, "storage-path", "s", "", "Override VM storage path")
}
