package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/detect"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
)

const (
	downloadImageChoice  = "__download__"
	customDownloadChoice = "__custom__"
)

var errDownloadSkipped = errors.New("download skipped")

// firstImage is the non-interactive image choice used by --yes.
func firstImage(imgPath string) (detect.ScannedImage, error) {
	images, err := detect.Scan(imgPath)
	if err != nil {
		return detect.ScannedImage{}, fmt.Errorf("scan images: %w", err)
	}
	if len(images) == 0 {
		return detect.ScannedImage{}, fmt.Errorf("no cloud images found in %s; run `kdeploy download` or add an image", imgPath)
	}
	return images[0], nil
}

// selectImage shows the Huh image picker and enters the download flow when the
// library is empty or the user chooses “Download a new image…”.
func selectImage(ctx context.Context, u *ui.UI, imgPath string) (detect.ScannedImage, error) {
	for {
		images, err := detect.Scan(imgPath)
		if err != nil {
			return detect.ScannedImage{}, fmt.Errorf("scan images: %w", err)
		}
		if len(images) == 0 {
			u.Warn("No cloud images found in %s", imgPath)
			if err := downloadImageInteractive(ctx, u, imgPath); err != nil {
				if errors.Is(err, errDownloadSkipped) {
					return detect.ScannedImage{}, fmt.Errorf("no image downloaded")
				}
				return detect.ScannedImage{}, err
			}
			continue
		}

		choice := images[0].Path
		height := len(images) + 3
		if height < 5 {
			height = 5
		}
		if height > 12 {
			height = 12
		}
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Image").
					Description("Choose the cloud image to boot.").
					Options(imageOptions(images)...).
					Value(&choice).
					Height(height),
			).Title("Image library"),
		)
		if err := u.RunForm(ctx, form); err != nil {
			return detect.ScannedImage{}, err
		}
		if choice == downloadImageChoice {
			if err := downloadImageInteractive(ctx, u, imgPath); err != nil {
				if errors.Is(err, errDownloadSkipped) {
					continue
				}
				return detect.ScannedImage{}, err
			}
			continue
		}
		if image, ok := findImage(images, choice); ok {
			return image, nil
		}
		return detect.ScannedImage{}, fmt.Errorf("selected image is no longer available: %s", choice)
	}
}

func imageOptions(images []detect.ScannedImage) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(images)+1)
	for _, image := range images {
		options = append(options, huh.NewOption(
			fmt.Sprintf("%s  (%s)", image.Filename, imageOSLabel(image.OS)),
			image.Path,
		))
	}
	return append(options, huh.NewOption("Download a new image…", downloadImageChoice))
}

func findImage(images []detect.ScannedImage, path string) (detect.ScannedImage, bool) {
	for _, image := range images {
		if image.Path == path {
			return image, true
		}
	}
	return detect.ScannedImage{}, false
}

func downloadImageInteractive(ctx context.Context, u *ui.UI, imgPath string) error {
	catalog := buildCatalog()
	if len(catalog) == 0 {
		return fmt.Errorf("no catalog images available")
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
				Options(options...).
				Value(&choice).
				Height(12),
		),
	)
	if err := u.RunForm(ctx, form); err != nil {
		return err
	}

	var rawURL string
	if choice == customDownloadChoice {
		form := huh.NewForm(
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
			return err
		}
	} else {
		for _, entry := range catalog {
			if entry.Key == choice {
				rawURL = entry.URL
				break
			}
		}
		if rawURL == "" {
			return fmt.Errorf("unknown catalog image %q", choice)
		}
	}

	filename, err := imageFilenameFromURL(rawURL)
	if err != nil {
		return err
	}
	dest := filepath.Join(imgPath, filename)
	overwrite := false
	if _, err := os.Stat(dest); err == nil {
		overwrite = false
		if err := u.Confirm(ctx, fmt.Sprintf("%s already exists. Overwrite it?", filename), &overwrite); err != nil {
			return err
		}
		if !overwrite {
			u.Muted("Download skipped.")
			return errDownloadSkipped
		}
	}

	u.Status("Downloading %s", filename)
	u.Muted("From: %s", rawURL)
	u.Muted("To:   %s", dest)
	if err := downloadFileWithOptions(rawURL, imgPath, hypervisor.NeedsPrivilegedPath(imgPath), overwrite); err != nil {
		return err
	}
	u.Success("Downloaded %s", filename)
	return nil
}

// promptResources collects the resource and access values. The form is
// prefilled from resolved options so Edit retains previous answers.
func promptResources(ctx context.Context, u *ui.UI, opts DeployOptions) (DeployOptions, error) {
	ram := opts.RAM
	cpus := opts.CPUs
	size := opts.Size
	network := opts.Network
	password := opts.Password
	packages := strings.Join(opts.Packages, " ")

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Memory (MB)").
				Value(&ram).
				Validate(ValidateRAM),
			huh.NewInput().
				Title("vCPUs").
				Value(&cpus).
				Validate(ValidateCPUs),
			huh.NewInput().
				Title("Disk size").
				Description("Use a G, M, or T suffix, for example 20G.").
				Value(&size).
				Validate(ValidateDiskSize),
		).Title("Resources"),
		huh.NewGroup(
			huh.NewInput().
				Title("Network").
				Description("Libvirt network name; add ,mac=AA:BB:CC:DD:EE:FF for a fixed MAC.").
				Value(&network).
				Validate(ValidateNetwork),
			huh.NewInput().
				Title("Root password").
				Description("Used for the root account in the cloud-init seed.").
				EchoMode(huh.EchoModePassword).
				Value(&password).
				Validate(ui.NotEmpty("root password")),
			huh.NewInput().
				Title("Packages").
				Description("Space-separated packages; leave empty or enter none for no extra packages.").
				Value(&packages).
				Validate(func(value string) error {
					return ValidatePackages(parsePackages(value))
				}),
		).Title("Access"),
	)
	if err := u.RunForm(ctx, form); err != nil {
		return DeployOptions{}, err
	}

	opts.RAM = strings.TrimSpace(ram)
	opts.CPUs = strings.TrimSpace(cpus)
	opts.Size = strings.TrimSpace(size)
	opts.Network = strings.TrimSpace(network)
	opts.Password = password
	opts.Packages = parsePackages(packages)
	return opts, nil
}

func parsePackages(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") {
		return nil
	}
	return strings.Fields(value)
}

// reviewDeploy renders the review panel and loops until the user creates,
// cancels, edits settings, or changes the image.
func reviewDeploy(ctx context.Context, u *ui.UI, opts DeployOptions, image detect.ScannedImage, existing bool) (DeployOptions, detect.ScannedImage, error) {
	for {
		renderReview(u, opts, image, existing)
		action := "create"
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Next").
					Options(reviewActionOptions()...).
					Value(&action),
			),
		)
		if err := u.RunForm(ctx, form); err != nil {
			return DeployOptions{}, detect.ScannedImage{}, err
		}

		switch action {
		case "create":
			return opts, image, nil
		case "edit":
			updated, err := promptResources(ctx, u, opts)
			if err != nil {
				return DeployOptions{}, detect.ScannedImage{}, err
			}
			opts = updated
		case "image":
			updated, err := selectImage(ctx, u, opts.ImagePath)
			if err != nil {
				return DeployOptions{}, detect.ScannedImage{}, err
			}
			image = updated
		case "cancel":
			return DeployOptions{}, detect.ScannedImage{}, ui.ErrCancelled
		default:
			return DeployOptions{}, detect.ScannedImage{}, fmt.Errorf("unknown review action %q", action)
		}
	}
}

func renderReview(u *ui.UI, opts DeployOptions, image detect.ScannedImage, existing bool) {
	title := "Review"
	if existing {
		title = "Review — replacement"
	}
	u.Panel(title, reviewRows(opts, image)...)
	if existing {
		u.Warn("VM %q already exists; it will be destroyed only after you choose Create.", opts.Name)
	}
}

func reviewRows(opts DeployOptions, image detect.ScannedImage) []string {
	rows := []string{
		"Name", opts.Name,
		"Image", image.Filename,
		"OS", imageOSLabel(image.OS),
		"Resources", fmt.Sprintf("%s MB RAM · %s vCPU · %s disk", opts.RAM, opts.CPUs, opts.Size),
		"Network", opts.Network,
		"Storage", opts.StoragePath,
		"SSH", fmt.Sprintf("%s, %s-root", opts.Name, opts.Name),
	}
	if len(opts.Packages) > 0 {
		rows = append(rows, "Packages", strings.Join(opts.Packages, " "))
	}
	return rows
}

func reviewActionOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("Create VM", "create"),
		huh.NewOption("Edit settings", "edit"),
		huh.NewOption("Change image", "image"),
		huh.NewOption("Cancel", "cancel"),
	}
}

func imageOSLabel(info detect.ImageInfo) string {
	if info.Distro != "" && info.Release != "" && info.Distro != "unknown" {
		return titleWord(info.Distro) + " " + info.Release
	}
	if info.OSVariant != "" {
		return info.OSVariant
	}
	return "unknown"
}

func titleWord(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
