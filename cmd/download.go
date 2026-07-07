package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hazembook/kdeploy/pkg/config"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/spf13/cobra"
)

type catalogEntry struct {
	Key  string
	Name string
	URL  string
}

func buildCatalog() []catalogEntry {
	arch := runtime.GOARCH
	rockyArch := "x86_64"
	ubuntuArch := "amd64"
	debianArch := "amd64"
	if arch == "arm64" {
		rockyArch = "aarch64"
		ubuntuArch = "arm64"
		debianArch = "arm64"
	}
	return []catalogEntry{
		{"rocky10", "Rocky Linux 10 (latest)", fmt.Sprintf("https://dl.rockylinux.org/pub/rocky/10/images/%s/Rocky-10-GenericCloud-Base.latest.%s.qcow2", rockyArch, rockyArch)},
		{"rocky9", "Rocky Linux 9 (latest)", fmt.Sprintf("https://dl.rockylinux.org/pub/rocky/9/images/%s/Rocky-9-GenericCloud-Base.latest.%s.qcow2", rockyArch, rockyArch)},
		{"rocky8", "Rocky Linux 8 (latest)", fmt.Sprintf("https://dl.rockylinux.org/pub/rocky/8/images/%s/Rocky-8-GenericCloud-Base.latest.%s.qcow2", rockyArch, rockyArch)},
		{"ubuntu2404", "Ubuntu 24.04 LTS (Noble)", fmt.Sprintf("https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-%s.img", ubuntuArch)},
		{"ubuntu2204", "Ubuntu 22.04 LTS (Jammy)", fmt.Sprintf("https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-%s.img", ubuntuArch)},
		{"ubuntu2004", "Ubuntu 20.04 LTS (Focal)", fmt.Sprintf("https://cloud-images.ubuntu.com/focal/current/focal-server-cloudimg-%s.img", ubuntuArch)},
		{"debian13", "Debian 13 (Trixie)", fmt.Sprintf("https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-%s.qcow2", debianArch)},
		{"debian12", "Debian 12 (Bookworm)", fmt.Sprintf("https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-%s.qcow2", debianArch)},
		{"debian11", "Debian 11 (Bullseye)", fmt.Sprintf("https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-generic-%s.qcow2", debianArch)},
	}
}

var downloadCmd = &cobra.Command{
	Use:   "download [key|url]",
	Short: "Download a cloud image",
	Long: `Download a cloud image from the built-in catalog or a custom URL.

Available keys: rocky10, rocky9, rocky8, ubuntu2404, ubuntu2204,
ubuntu2004, debian13, debian12, debian11

Use --url to download from a custom URL, or pass a key/URL as an argument.
Run without arguments for an interactive picker.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listFlag, _ := cmd.Flags().GetBool("list")
		urlFlag, _ := cmd.Flags().GetString("url")

		if listFlag {
			printCatalog()
			return nil
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		imgPath := cfg.ImagePath
		if imagePath != "" {
			imgPath = imagePath
		}
		os.MkdirAll(imgPath, 0755)
		useSudo := hypervisor.NeedsPrivilegedPath(imgPath)

		catalog := buildCatalog()
		var entry *catalogEntry
		var customURL string

		if len(args) == 1 {
			arg := args[0]
			if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
				customURL = arg
			} else {
				for i := range catalog {
					if catalog[i].Key == arg {
						entry = &catalog[i]
						break
					}
				}
				if entry == nil {
					fmt.Printf("Unknown key %q. Available:\n", arg)
					printCatalog()
					return nil
				}
			}
		} else if urlFlag != "" {
			customURL = urlFlag
		} else {
			return pickAndDownload(catalog, imgPath, useSudo)
		}

		if entry != nil {
			return downloadFile(entry.URL, imgPath, useSudo)
		}
		return downloadFile(customURL, imgPath, useSudo)
	},
}

func printCatalog() {
	fmt.Println("Available images:")
	for _, e := range buildCatalog() {
		fmt.Printf("  %-14s %s\n", e.Key, e.Name)
	}
}

func pickAndDownload(catalog []catalogEntry, imgPath string, useSudo bool) error {
	fmt.Println("Available images:")
	for i, e := range catalog {
		fmt.Printf("  %2d) %-14s %s\n", i+1, e.Key, e.Name)
	}
	fmt.Printf("  %2d) Custom URL\n", len(catalog)+1)
	choice := promptValue("Select image", "1")
	var idx int
	if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(catalog)+1 {
		return fmt.Errorf("invalid selection")
	}
	if idx == len(catalog)+1 {
		url := promptValue("Enter URL", "")
		if url == "" {
			return fmt.Errorf("no URL provided")
		}
		return downloadFile(url, imgPath, useSudo)
	}
	return downloadFile(catalog[idx-1].URL, imgPath, useSudo)
}

func downloadFile(url, destDir string, useSudo bool) error {
	filename := filepath.Base(url)
	dest := filepath.Join(destDir, filename)

	if _, err := os.Stat(dest); err == nil {
		overwrite := promptValue(fmt.Sprintf("📥 %s already exists, overwrite?", filename), "n")
		if !strings.EqualFold(overwrite, "y") {
			fmt.Println("   Skipped.")
			return nil
		}
	}

	fmt.Printf("📥 Downloading: %s\n", filename)
	fmt.Printf("   From: %s\n", url)
	fmt.Printf("   To:   %s\n", dest)

	var cmd *exec.Cmd
	if commandExists("wget") {
		args := []string{"wget", "-c", "--progress=bar:force", url, "-O", dest}
		if useSudo {
			cmd = exec.Command("sudo", args...)
		} else {
			cmd = exec.Command(args[0], args[1:]...)
		}
	} else if commandExists("curl") {
		args := []string{"curl", "-L", "-C", "-", "--progress-bar", url, "-o", dest}
		if useSudo {
			cmd = exec.Command("sudo", args...)
		} else {
			cmd = exec.Command(args[0], args[1:]...)
		}
	} else {
		return fmt.Errorf("neither wget nor curl found — install one to download images")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if useSudo {
		exec.Command("sudo", "chmod", "644", dest).Run()
	} else {
		os.Chmod(dest, 0644)
	}

	fmt.Println("   ✅ Download complete.")
	return nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().BoolP("list", "l", false, "List available images")
	downloadCmd.Flags().String("url", "", "Download from a custom URL")
}
