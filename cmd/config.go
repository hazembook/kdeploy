package cmd

import (
	"fmt"
	"os"

	"github.com/hazembook/kdeploy/pkg/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		home, _ := os.UserHomeDir()
		path := home + "/.config/kdeploy.conf"
		if _, err := os.Stat(path); err == nil && !force {
			return fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
		}

		cfg := config.DefaultConfig()
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("📝 Config created at %s\n", path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		fmt.Println("📋 Configuration:")
		fmt.Printf("   Image path:    %s\n", cfg.ImagePath)
		fmt.Printf("   Storage path:  %s\n", cfg.StoragePath)
		fmt.Printf("   Default size:  %s\n", cfg.DefaultSize)
		fmt.Printf("   Default RAM:   %s\n", cfg.DefaultRAM)
		fmt.Printf("   Default CPUs:  %s\n", cfg.DefaultCPUs)
		fmt.Printf("   Default pass:  %s\n", cfg.DefaultPass)
		return nil
	},
}

func init() {
	configInitCmd.Flags().BoolP("force", "f", false, "Overwrite existing config")
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)

	rootCmd.PersistentFlags().Bool("reconfig", false, "Recreate default config and exit")
	rootCmd.PersistentFlags().Bool("show-config", false, "Show current config and exit")
}
