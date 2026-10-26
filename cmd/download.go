package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/huh/v2"
	"github.com/hazembook/kdeploy/internal/ui"
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
Run without arguments for an interactive picker. --yes requires a key or URL.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New()
		listFlag, _ := cmd.Flags().GetBool("list")
		urlFlag, _ := cmd.Flags().GetString("url")

		if listFlag {
			renderDownloadCatalog(u)
			return nil
		}

		if len(args) == 0 && urlFlag == "" {
			if skipConfirm {
				return fmt.Errorf("download --yes needs an image key or URL; run `kdeploy download --list` to see available images")
			}
			if !u.CanPrompt() {
				return fmt.Errorf("%w: pass an image key or URL, or use --list", ui.ErrNonInteractive)
			}
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		imgPath := config.ExpandPath(cfg.ImagePath)
		if imagePath != "" {
			imgPath = config.ExpandPath(imagePath)
		}
		if err := ensureDir(imgPath); err != nil {
			return fmt.Errorf("image library: %w", err)
		}
		useSudo := hypervisor.NeedsPrivilegedPath(imgPath)

		catalog := buildCatalog()
		var rawURL string
		switch {
		case len(args) == 1:
			rawURL, err = resolveDownloadArg(catalog, args[0])
			if err != nil {
				return err
			}
		case urlFlag != "":
			rawURL = strings.TrimSpace(urlFlag)
		default:
			rawURL, err = pickDownloadImage(cmd.Context(), u, catalog)
			if err != nil {
				return err
			}
		}
		rawURL = strings.TrimSpace(rawURL)

		filename, err := imageFilenameFromURL(rawURL)
		if err != nil {
			return err
		}
		dest := filepath.Join(imgPath, filename)
		if _, err := confirmDownloadOverwrite(cmd.Context(), u, dest, filename); err != nil {
			return err
		}
		if err := downloadFileWithOptions(rawURL, imgPath, useSudo); err != nil {
			return err
		}

		u.KV("Downloaded", dest)
		return nil
	},
}

func renderDownloadCatalog(u *ui.UI) {
	entries := buildCatalog()
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{entry.Key, entry.Name})
	}
	u.Heading("Available images")
	u.Table([]string{"Key", "Image"}, rows)
}

func resolveDownloadArg(catalog []catalogEntry, value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "://") {
		return value, nil
	}
	for _, entry := range catalog {
		if entry.Key == value {
			return entry.URL, nil
		}
	}
	return "", fmt.Errorf("unknown image key %q; run `kdeploy download --list` to see available images", value)
}

func pickDownloadImage(ctx context.Context, u *ui.UI, catalog []catalogEntry) (string, error) {
	if len(catalog) == 0 {
		return "", fmt.Errorf("no catalog images available")
	}

	choice := catalog[0].Key
	options := make([]huh.Option[string], 0, len(catalog)+1)
	for _, entry := range catalog {
		options = append(options, huh.NewOption(
			fmt.Sprintf("%s  (%s)", entry.Name, entry.Key),
			entry.Key,
		))
	}
	options = append(options, huh.NewOption("Custom URL…", customDownloadChoice))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Download a cloud image").
				Description("Choose a catalog image or enter a custom URL.").
				Options(options...).
				Value(&choice).
				Height(12),
		).Title("Catalog"),
	)
	if err := u.RunForm(ctx, form); err != nil {
		return "", err
	}

	if choice != customDownloadChoice {
		for _, entry := range catalog {
			if entry.Key == choice {
				return entry.URL, nil
			}
		}
		return "", fmt.Errorf("unknown catalog image %q", choice)
	}

	var rawURL string
	form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Image URL").
				Description("An http or https URL to a .qcow2 or .img file.").
				Value(&rawURL).
				Validate(func(value string) error {
					_, err := imageFilenameFromURL(value)
					return err
				}),
		),
	)
	if err := u.RunForm(ctx, form); err != nil {
		return "", err
	}
	return strings.TrimSpace(rawURL), nil
}

func confirmDownloadOverwrite(ctx context.Context, u *ui.UI, dest, filename string) (bool, error) {
	if _, err := os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("inspect %s: %w", dest, err)
	}
	if skipConfirm {
		return true, nil
	}
	if !u.CanPrompt() {
		return false, fmt.Errorf("%w: %s already exists; use --yes to overwrite", ui.ErrNonInteractive, filename)
	}

	overwrite := false
	if err := u.Confirm(ctx, fmt.Sprintf("%s already exists. Overwrite?", filename), &overwrite); err != nil {
		return false, err
	}
	if !overwrite {
		return false, fmt.Errorf("download cancelled")
	}
	return true, nil
}

// downloadFileWithOptions downloads rawURL into destDir. Callers decide the
// overwrite policy first; the native downloader keeps its resume behavior.
func downloadFileWithOptions(rawURL, destDir string, useSudo bool) error {
	rawURL = strings.TrimSpace(rawURL)
	filename, err := imageFilenameFromURL(rawURL)
	if err != nil {
		return err
	}
	dest := filepath.Join(destDir, filename)
	if _, err := os.Stat(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s: %w", dest, err)
	}

	fmt.Fprintf(os.Stderr, "Downloading: %s\n", filename)
	fmt.Fprintf(os.Stderr, "   From: %s\n", rawURL)
	fmt.Fprintf(os.Stderr, "   To:   %s\n", dest)

	var cmd *exec.Cmd
	if commandExists("wget") {
		args := []string{"wget", "-c", "--progress=bar:force", rawURL, "-O", dest}
		if useSudo {
			cmd = exec.Command("sudo", args...)
		} else {
			cmd = exec.Command(args[0], args[1:]...)
		}
	} else if commandExists("curl") {
		args := []string{"curl", "-L", "-C", "-", "--progress-bar", rawURL, "-o", dest}
		if useSudo {
			cmd = exec.Command("sudo", args...)
		} else {
			cmd = exec.Command(args[0], args[1:]...)
		}
	} else {
		return fmt.Errorf("neither wget nor curl found — install one to download images")
	}

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if useSudo {
		exec.Command("sudo", "chmod", "644", dest).Run()
	} else {
		os.Chmod(dest, 0644)
	}

	fmt.Fprintln(os.Stderr, "   Download complete.")
	return nil
}

func imageFilenameFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("image URL must be an http or https URL: %q", rawURL)
	}
	filename := path.Base(parsed.EscapedPath())
	if filename == "" || filename == "." || filename == "/" {
		return "", fmt.Errorf("image URL must include a filename: %q", rawURL)
	}
	unescaped, err := url.PathUnescape(filename)
	if err != nil {
		return "", fmt.Errorf("image URL contains an invalid filename: %q", rawURL)
	}
	filename = unescaped
	if filename == "" || filename == "." || filename == ".." || strings.ContainsAny(filename, `/\`) {
		return "", fmt.Errorf("image URL must include a plain filename: %q", rawURL)
	}
	if strings.ContainsRune(filename, 0) {
		return "", fmt.Errorf("image URL must include a plain filename: %q", rawURL)
	}
	return filename, nil
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
