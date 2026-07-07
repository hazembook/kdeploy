package cmd

import (
	"fmt"

	"github.com/hazembook/kdeploy/pkg/check"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check system dependencies",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		autoInstall, _ := cmd.Flags().GetBool("install")

		fmt.Println("🔍 Checking dependencies...")
		results := check.RequiredTools()

		allPassed := true
		for _, r := range results {
			if r.Passed {
				fmt.Printf("   ✅ %s\n", r.Name)
			} else {
				allPassed = false
				fmt.Printf("   ❌ %s\n", r.Name)
			}
		}

		if allPassed {
			fmt.Println("\n✅ All dependencies satisfied.")
			return nil
		}

		if !autoInstall {
			fmt.Println("\n⚠️  Some dependencies are missing.")
			fmt.Println("   Run with --install to auto-install, or install manually.")
			return nil
		}

		distro := check.DetectDistro()
		pkgMgr := check.PackageManager(distro)
		pkgs := check.DistroPackageList(distro)

		if pkgs == "" {
			fmt.Printf("⚠️  No package list for distro %q. Install manually:\n", distro)
			fmt.Printf("   %s\n", check.DistroPackageList("arch"))
			return nil
		}

		fmt.Printf("\n📦 Detected: %s (%s)\n", distro, pkgMgr)
		fmt.Printf("   Packages: %s\n", pkgs)
		fmt.Println()

		if !skipConfirm && !confirmPrompt("   Install now?") {
			fmt.Println("   Skipped.")
			return nil
		}

		fmt.Println("   Installing...")
		if err := check.InstallDependencies(distro, pkgMgr); err != nil {
			return fmt.Errorf("install: %w", err)
		}
		fmt.Println("✅ Dependencies installed.")
		return nil
	},
}

func init() {
	checkCmd.Flags().Bool("install", false, "Auto-install missing dependencies")
	rootCmd.AddCommand(checkCmd)
}
