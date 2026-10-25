package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/internal/ui"
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
	Short: "Create a default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		path := effectiveConfigPath()
		if config.Exists(path) && !force {
			return fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
		}

		u := ui.New()
		if _, err := runSetup(cmd.Context(), u, SetupOptions{
			Path:  path,
			Write: true,
			Force: force,
			Reset: true,
		}); err != nil {
			return err
		}
		u.Println("Config created at " + path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showConfig(ui.New())
	},
}

func showConfig(u *ui.UI) error {
	path := effectiveConfigPath()
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	u.Heading("Configuration")
	u.Table([]string{"Setting", "Value"}, [][]string{
		{"File", path},
		{"Image path", cfg.ImagePath},
		{"Storage path", cfg.StoragePath},
		{"Default size", cfg.DefaultSize},
		{"Default RAM", cfg.DefaultRAM},
		{"Default CPUs", cfg.DefaultCPUs},
		{"Default pass", u.MaskSecret(cfg.DefaultPass)},
	})
	return nil
}

func effectiveConfigPath() string {
	path := config.ExpandPath(cfgFile)
	if path == "" {
		path = config.DefaultPath()
	}
	return path
}

func init() {
	configInitCmd.Flags().BoolP("force", "f", false, "Overwrite existing config")
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)

	rootCmd.PersistentFlags().Bool("reconfig", false, "Reconfigure image/storage paths and exit")
	rootCmd.PersistentFlags().Bool("show-config", false, "Show current config and exit")
}
