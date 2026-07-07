package cmd

import (
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

		cmdSudo := hypervisor.NeedsSudo()                  // for virsh/virt-install
		fileSudo := hypervisor.NeedsPrivilegedPath(storPath) // for qemu-img, cloud-localds, rm

		// 2. Pre-flight checks
		fmt.Println("🔍 Running pre-flight checks...")
		for _, r := range check.All(storPath, diskSpaceMinKB) {
			if r.Passed {
				fmt.Printf("   ✅ %s\n", r.Name)
			} else {
				fmt.Printf("   ❌ %s: %s\n", r.Name, r.Error)
				return fmt.Errorf("pre-flight check failed: %s", r.Name)
			}
		}
		keyPath, kr := check.SSHKey()
		if !kr.Passed {
			return fmt.Errorf("pre-flight: %s", kr.Error)
		}
		fmt.Printf("   ✅ %s (%s)\n", kr.Name, keyPath)

		// 3. Scan images
		images, err := detect.Scan(imgPath)
		if err != nil {
			return fmt.Errorf("scan images: %w", err)
		}
		if len(images) == 0 {
			return fmt.Errorf("no cloud images found in %s", imgPath)
		}
		selected := images[0]
		fmt.Printf("📦 Selected image: %s (%s)\n", selected.Filename, selected.OS.OSVariant)

		// 4. Generate password hash
		pass := cfg.DefaultPass
		hash, err := iso.HashPassword(pass)
		if err != nil {
			return fmt.Errorf("password hash: %w", err)
		}

		// 5. Generate cloud-init ISO
		cloudData := &iso.CloudInitData{
			Hostname:     name,
			InstanceID:   fmt.Sprintf("%s-%d", name, time.Now().Unix()),
			SSHPublicKey: readPubKey(keyPath),
			PasswordHash: hash,
			Username:     os.Getenv("USER"),
		}
		seedPath := filepath.Join(storPath, name+"-seed.iso")
		fmt.Println("🔧 Generating cloud-init...")
		if err := iso.GenerateSeedISO(cloudData, seedPath, fileSudo); err != nil {
			return fmt.Errorf("cloud-init iso: %w", err)
		}

		// 6. Create overlay disk
		overlayPath := filepath.Join(storPath, name+".qcow2")
		fmt.Printf("💾 Creating overlay image (%s)...\n", size)
		if err := storage.CreateOverlay(selected.Path, overlayPath, size, selected.OS.OSVariant, fileSudo); err != nil {
			return fmt.Errorf("overlay: %w", err)
		}

		// 7. Deploy VM
		fmt.Println("🚀 Booting VM...")
		vmCfg := &hypervisor.VMConfig{
			Name:      name,
			RAM:       vmRAM,
			CPUs:      vmCPUs,
			DiskPath:  overlayPath,
			ISOPath:   seedPath,
			Network:   vmNetwork,
			OSVariant: selected.OS.OSVariant,
		}
		if err := hypervisor.Deploy(vmCfg, cmdSudo); err != nil {
			return fmt.Errorf("deploy: %w", err)
		}

		// 8. Wait for IP
		fmt.Println("⏳ Waiting for IP address...")
		ip, err := hypervisor.WaitForIP(name, netName(vmNetwork), cmdSudo, 90*time.Second)
		if err != nil {
			return fmt.Errorf("ip discovery: %w", err)
		}
		fmt.Printf("   ✅ IP: %s\n", ip)

		// 9. Clear known hosts entry
		exec.Command("ssh-keygen", "-R", ip).Run()

		// 10. Wait for SSH
		fmt.Println("⏳ Waiting for SSH...")
		if err := ssh.WaitForSSH(ip, 60*time.Second); err != nil {
			fmt.Printf("   ⚠️  %s\n", err)
		} else {
			fmt.Println("   ✅ SSH ready")
		}

		// 11. Write SSH config
		incDir, _ := ssh.IncludeDir()
		os.MkdirAll(incDir, 0700)
		ssh.EnsureInclude(filepath.Join(os.Getenv("HOME"), ".ssh", "config"))
		if err := ssh.WriteHostConfig(name, ip, os.Getenv("USER"), keyPath, incDir); err != nil {
			return fmt.Errorf("ssh config: %w", err)
		}

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
