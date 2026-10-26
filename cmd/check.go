package cmd

import (
	"fmt"
	"strings"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/check"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check system dependencies",
	Long: `Checks the command-line tools required by kdeploy.

The command exits non-zero when any dependency is missing. Use --install to
install the missing tools for the detected distribution.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		autoInstall, _ := cmd.Flags().GetBool("install")

		results := check.RequiredTools()
		renderCheckResults(u, results)
		if err := checkExit(results); err != nil {
			if !autoInstall {
				u.Muted("Install them with `kdeploy check --install` or manually.")
				return err
			}
			if !skipConfirm {
				if err := u.RequireInteractive("install confirmation"); err != nil {
					return err
				}
			}

			distro := check.DetectDistro()
			pkgMgr := check.PackageManager(distro)
			pkgs := check.DistroPackageList(distro)
			if pkgs == "" {
				return fmt.Errorf("no package list for distro %q; install the missing tools manually", distro)
			}
			if pkgMgr == "unknown" || !strings.Contains(pkgMgr, " ") {
				return fmt.Errorf("no supported package manager found for distro %q; install the missing tools manually", distro)
			}

			u.Section("Installing dependencies")
			u.KV("Distro", distro, "Package manager", pkgMgr, "Packages", pkgs)

			if !skipConfirm {
				install := false
				if err := u.Confirm(cmd.Context(), fmt.Sprintf("Install missing dependencies with %s?", pkgMgr), &install); err != nil {
					return err
				}
				if !install {
					u.Muted("Install declined.")
					return err
				}
			}

			if err := check.InstallDependencies(distro, pkgMgr); err != nil {
				return fmt.Errorf("install: %w", err)
			}

			results = check.RequiredTools()
			renderCheckResults(u, results)
			if err := checkExit(results); err != nil {
				return err
			}
			u.Success("Dependencies installed")
		}
		return nil
	},
}

// checkExit is the pure exit decision for `check`. It is kept separate from
// rendering so the gate can be table-tested without touching the environment.
func checkExit(results []check.Result) error {
	var failures []string
	passed := 0
	for _, result := range results {
		if result.Passed {
			passed++
			continue
		}
		detail := result.Error
		if detail == "" {
			detail = "failed"
		}
		failures = append(failures, fmt.Sprintf("%s: %s", result.Name, detail))
	}
	if len(failures) == 0 {
		return nil
	}
	return fmt.Errorf("%d/%d checks passed; %s", passed, len(results), strings.Join(failures, "; "))
}

func renderCheckResults(u *ui.UI, results []check.Result) {
	rows := make([][]string, 0, len(results))
	passed := 0
	for _, result := range results {
		status := "fail"
		if result.Passed {
			status = "ok"
			passed++
		}
		rows = append(rows, []string{result.Name, status, result.Error})
	}

	u.Heading("Dependency checks")
	u.Table([]string{"Check", "Status", "Detail"}, rows)
	u.KV("Passed", fmt.Sprintf("%d/%d checks", passed, len(results)))
}

func init() {
	checkCmd.Flags().Bool("install", false, "Install missing dependencies")
	rootCmd.AddCommand(checkCmd)
}
