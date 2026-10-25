package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hazembook/kdeploy/pkg/check"
	"github.com/hazembook/kdeploy/pkg/config"
	"github.com/hazembook/kdeploy/pkg/detect"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/hazembook/kdeploy/pkg/iso"
	"github.com/hazembook/kdeploy/pkg/ssh"
	"github.com/hazembook/kdeploy/pkg/storage"
	"github.com/spf13/cobra"
)

func pickImage(images []detect.ScannedImage, imgPath string) *detect.ScannedImage {
	fmt.Println("\n📦 Available images:")
	for i, img := range images {
		fmt.Printf("  %2d) %-50s %s\n", i+1, img.Filename, img.OS.OSVariant)
	}
	fmt.Printf("  %2d) 📥 Download a new image\n", len(images)+1)
	choice := promptValue(fmt.Sprintf("Select image [1-%d]", len(images)+1), "1")
	var idx int
	if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil {
		return &images[0]
	}
	if idx == len(images)+1 {
		if err := setupImagePicker(imgPath); err != nil {
			fmt.Printf("   ❌ %v\n", err)
			return &images[0]
		}
		rescanned, _ := detect.Scan(imgPath)
		if len(rescanned) == 0 {
			return &images[0]
		}
		return pickImage(rescanned, imgPath)
	}
	if idx < 1 || idx > len(images) {
		return &images[0]
	}
	return &images[idx-1]
}

func setupImagePicker(imgPath string) error {
	fmt.Println("\n📥 Image downloader")
	fmt.Println("   Choose a source:")
	fmt.Println("   1) Built-in catalog")
	fmt.Println("   2) Custom URL")
	choice := promptValue("   Select", "1")
	if choice == "2" {
		url := promptValue("   Enter URL", "")
		if url == "" {
			return fmt.Errorf("no URL provided")
		}
		useSudo := hypervisor.NeedsPrivilegedPath(imgPath)
		return downloadFile(url, imgPath, useSudo)
	}
	return pickAndDownload(buildCatalog(), imgPath, hypervisor.NeedsPrivilegedPath(imgPath))
}

func promptValue(prompt, defaultVal string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultVal)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	val := strings.TrimSpace(scanner.Text())
	if val == "" {
		return defaultVal
	}
	return val
}

func confirmPrompt(msg string) bool {
	fmt.Printf("   %s [Y/n] ", msg)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := strings.TrimSpace(scanner.Text())
	return answer == "" || strings.EqualFold(answer, "y")
}

const diskSpaceMinKB = 20 * 1024 * 1024 // 20GB

func netName(network string) string {
	return strings.Split(network, ",")[0]
}

var deployCmd = &cobra.Command{
	Use:   "deploy <vm_name> [disk_size]",
	Short: "Deploy a new VM",
	Long: `Creates and boots a new KVM virtual machine from a cloud image.
VM name is required. Optional disk size (default: 20G).`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		size := vmSize

		var cleanupFiles []string
		defer func() {
			for _, f := range cleanupFiles {
				sudoRemove(f)
			}
		}()
		if len(args) > 1 {
			size = args[1]
		}

		// 1. Load config
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		imgPath := cfg.ImagePath
		storPath := cfg.StoragePath
		if imagePath != "" {
			imgPath = imagePath
		}
		if storagePath != "" {
			storPath = storagePath
		}

		cmdSudo := hypervisor.NeedsSudo()                    // for virsh/virt-install
		fileSudo := hypervisor.NeedsPrivilegedPath(storPath) // for qemu-img, cloud-localds, rm

		// 2. Pre-flight checks
		fmt.Println("🔍 Running pre-flight checks...")

		deployTools := check.RequiredTools()
		allToolLines := append(check.All(storPath, diskSpaceMinKB), deployTools...)

		for _, r := range allToolLines {
			if r.Passed {
				fmt.Printf("   ✅ %s\n", r.Name)
			} else {
				fmt.Printf("   ❌ %s: %s\n", r.Name, r.Error)
				return fmt.Errorf("pre-flight check failed: %s", r.Name)
			}
		}

		// 2a. Auto-fix SSH key if missing
		keyPath, err := check.EnsureSSHKey()
		if err != nil {
			return fmt.Errorf("ssh key: %w", err)
		}
		fmt.Printf("   ✅ SSH key (%s)\n", keyPath)

		// 2b. Ensure default network is active
		netResult := check.DefaultNetwork(cmdSudo)
		if netResult.Passed {
			fmt.Println("   ✅ Default network active")
		} else {
			fmt.Printf("   ⚙️  %s...\n", netResult.Error)
			if err := check.EnsureDefaultNetwork(cmdSudo); err != nil {
				return fmt.Errorf("default network: %w", err)
			}
			fmt.Println("   ✅ Default network ready")
		}

		// 2c. Firewall rule for custom bridge networks
		bridgeName := netName(vmNetwork)
		if bridgeName != "default" {
			fmt.Printf("   ⚙️  Configuring UFW for bridge %s...\n", bridgeName)
			check.EnsureFirewallRule(bridgeName, cmdSudo)
		}

		// 3. VM-exists check + cleanup
		if hypervisor.VMExists(name, cmdSudo) {
			fmt.Printf("   ⚠️  VM '%s' already exists — will be DESTROYED and recreated!\n", name)
			if !skipConfirm && !confirmPrompt("   Continue?") {
				return fmt.Errorf("aborted by user")
			}
			fmt.Println("🧹 Cleaning up old VM...")
			hypervisor.Destroy(name, cmdSudo)
			sudoRemove(filepath.Join(storPath, name+".qcow2"))
			sudoRemove(filepath.Join(storPath, name+"-seed.iso"))
			incDir, _ := ssh.IncludeDir()
			ssh.RemoveHostConfig(name, incDir)
		}

		// 4. Scan images
		images, err := detect.Scan(imgPath)
		if err != nil {
			return fmt.Errorf("scan images: %w", err)
		}
		if len(images) == 0 {
			if !skipConfirm {
				fmt.Println("   No images found — launching download prompt...")
				imgPath = cfg.ImagePath
				if imagePath != "" {
					imgPath = imagePath
				}
				if err := setupImagePicker(imgPath); err != nil {
					return err
				}
				images, _ = detect.Scan(imgPath)
			}
			if len(images) == 0 {
				return fmt.Errorf("no cloud images found in %s", imgPath)
			}
		}
		selected := images[0]
		if !skipConfirm {
			s := pickImage(images, imgPath)
			if s == nil {
				return fmt.Errorf("image selection cancelled")
			}
			selected = *s
		}
		fmt.Printf("📦 Selected image: %s (%s)\n", selected.Filename, selected.OS.OSVariant)

		// 5. Interactive prompts (gated by --yes)
		vmPass := cfg.DefaultPass
		vmRAMVal := vmRAM
		vmCPUsVal := vmCPUs
		var pkgList []string
		if !skipConfirm {
			promptPass := strings.TrimSpace(promptValue("🔑 VM password", vmPass))
			if promptPass != "" {
				vmPass = promptPass
			}
			vmRAMVal = promptValue("   RAM (MB)", vmRAM)
			vmCPUsVal = promptValue("   vCPUs", vmCPUs)
			pkgChoice := promptValue("📦 Install packages? (space-separated, or 'none')", "none")
			if pkgChoice != "" && !strings.EqualFold(pkgChoice, "none") {
				for _, p := range strings.Fields(pkgChoice) {
					pkgList = append(pkgList, strings.TrimSpace(p))
				}
			}
		}
		hash, err := iso.HashPassword(vmPass)
		if err != nil {
			return fmt.Errorf("password hash: %w", err)
		}

		// 6. Generate cloud-init ISO
		cloudData := &iso.CloudInitData{
			Hostname:     name,
			InstanceID:   fmt.Sprintf("%s-%d", name, time.Now().Unix()),
			SSHPublicKey: readPubKey(keyPath),
			PasswordHash: hash,
			Username:     os.Getenv("USER"),
			Packages:     pkgList,
		}
		seedPath := filepath.Join(storPath, name+"-seed.iso")
		cleanupFiles = append(cleanupFiles, seedPath)
		fmt.Println("🔧 Generating cloud-init...")
		if err := iso.GenerateSeedISO(cloudData, seedPath, fileSudo); err != nil {
			return fmt.Errorf("cloud-init iso: %w", err)
		}

		// 7. Detect backing format + create overlay
		backingFmt, err := storage.DetectFormat(selected.Path, fileSudo)
		if err != nil {
			return fmt.Errorf("detect backing format: %w", err)
		}
		overlayPath := filepath.Join(storPath, name+".qcow2")
		cleanupFiles = append(cleanupFiles, overlayPath)
		fmt.Printf("💾 Creating overlay image (%s)...\n", size)
		if err := storage.CreateOverlay(selected.Path, overlayPath, size, backingFmt, fileSudo); err != nil {
			return fmt.Errorf("overlay: %w", err)
		}

		// 8. Deploy VM
		fmt.Println("🚀 Booting VM...")
		vmCfg := &hypervisor.VMConfig{
			Name:      name,
			RAM:       vmRAMVal,
			CPUs:      vmCPUsVal,
			DiskPath:  overlayPath,
			ISOPath:   seedPath,
			Network:   vmNetwork,
			OSVariant: selected.OS.OSVariant,
		}
		if err := hypervisor.Deploy(vmCfg, cmdSudo); err != nil {
			return fmt.Errorf("deploy: %w", err)
		}

		// 9. Wait for IP
		fmt.Println("⏳ Waiting for IP address...")
		ip, err := hypervisor.WaitForIP(name, netName(vmNetwork), cmdSudo, 90*time.Second)
		if err != nil {
			return fmt.Errorf("ip discovery: %w", err)
		}
		fmt.Printf("   ✅ IP: %s\n", ip)

		// 10. Clear known hosts entry
		exec.Command("ssh-keygen", "-R", ip).Run()

		// 11. Wait for SSH
		fmt.Println("⏳ Waiting for SSH...")
		if err := ssh.WaitForSSH(ip, 60*time.Second); err != nil {
			fmt.Printf("   ⚠️  %s\n", err)
		} else {
			fmt.Println("   ✅ SSH ready")
		}

		// 12. Write SSH config
		incDir, _ := ssh.IncludeDir()
		os.MkdirAll(incDir, 0700)
		ssh.EnsureInclude(filepath.Join(os.Getenv("HOME"), ".ssh", "config"))
		if err := ssh.WriteHostConfig(name, ip, os.Getenv("USER"), keyPath, incDir); err != nil {
			return fmt.Errorf("ssh config: %w", err)
		}

		cleanupFiles = nil
		fmt.Println("\n✅ VM Ready!")
		fmt.Printf("   ssh %s\n", name)
		fmt.Printf("   ssh %s-root\n", name)
		return nil
	},
}

func readPubKey(keyPath string) string {
	data, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVarP(&vmSize, "size", "d", "20G", "Disk size (e.g. 20G, 50G)")
}
