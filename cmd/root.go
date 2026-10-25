package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/x/term"
	"github.com/hazembook/kdeploy/internal/ui"
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
			u := ui.New()
			if !u.CanPrompt() && !skipConfirm {
				return fmt.Errorf("%w: --reconfig needs prompts; use --yes to accept defaults", ui.ErrNonInteractive)
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			_, err := runSetup(ctx, u, SetupOptions{
				Path:        cfgFile,
				Write:       true,
				Force:       true,
				Interactive: u.CanPrompt() && !skipConfirm,
			})
			if err == nil {
				u.Println("Config written to " + effectiveConfigPath())
			}
			return err
		}
		if showCfg, _ := cmd.Flags().GetBool("show-config"); showCfg {
			return showConfig(ui.New())
		}
		return cmd.Help()
	},
}

func Execute(version string) {
	configureCommandGroups()
	err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version),
		fang.WithNotifySignal(os.Interrupt),
		fang.WithErrorHandler(handleError),
		fang.WithoutCompletions(),
		fang.WithoutManpage(),
	)
	if err == nil {
		return
	}
	if errors.Is(err, ui.ErrCancelled) {
		os.Exit(130)
	}
	os.Exit(1)
}

func handleError(w io.Writer, styles fang.Styles, err error) {
	if errors.Is(err, ui.ErrCancelled) {
		fmt.Fprintln(w, "Cancelled.")
		return
	}
	if !errorWriterIsTerminal(w) {
		fmt.Fprintln(w, err.Error())
		return
	}
	fang.DefaultErrorHandler(w, styles, err)
}

func errorWriterIsTerminal(w io.Writer) bool {
	writer, ok := w.(*colorprofile.Writer)
	if !ok {
		return false
	}
	file, ok := writer.Forward.(*os.File)
	return ok && term.IsTerminal(file.Fd())
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
	rootCmd.PersistentFlags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmations and use non-interactive defaults")
	rootCmd.PersistentFlags().StringVarP(&vmRAM, "ram", "r", "", "RAM in MB (default: config or 2048)")
	rootCmd.PersistentFlags().StringVarP(&vmCPUs, "cpu", "c", "", "Number of vCPUs (default: config or 2)")
	rootCmd.PersistentFlags().StringVarP(&vmNetwork, "network", "n", "", "Virtual network to use (default: default)")
	rootCmd.PersistentFlags().StringVarP(&imagePath, "image-path", "i", "", "Override image library path")
	rootCmd.PersistentFlags().StringVarP(&storagePath, "storage-path", "s", "", "Override VM storage path")
}
