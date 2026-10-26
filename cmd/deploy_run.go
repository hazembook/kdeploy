package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hazembook/kdeploy/internal/ui"
	"github.com/hazembook/kdeploy/pkg/check"
	"github.com/hazembook/kdeploy/pkg/detect"
	"github.com/hazembook/kdeploy/pkg/hypervisor"
	"github.com/hazembook/kdeploy/pkg/iso"
	"github.com/hazembook/kdeploy/pkg/ssh"
	"github.com/hazembook/kdeploy/pkg/storage"
)

// deployPreflight runs the full KVM/libvirt/disk/network/SSH preflight used by
// deploy (unlike the tool-focused `check` command).
func deployPreflight(u *ui.UI, opts DeployOptions) (keyPath string, cmdSudo bool, err error) {
	cmdSudo = hypervisor.NeedsSudo()
	u.Section("Pre-flight checks")

	results := append(check.All(opts.StoragePath, diskSpaceMinKB), check.RequiredTools()...)
	var failures []string
	for _, result := range results {
		if result.Passed {
			u.Success("%s", result.Name)
			continue
		}
		detail := result.Error
		if detail == "" {
			detail = "failed"
		}
		u.Failure("%s: %s", result.Name, detail)
		failures = append(failures, fmt.Sprintf("%s: %s", result.Name, detail))
	}
	if len(failures) > 0 {
		return "", cmdSudo, fmt.Errorf("pre-flight check failed: %s", strings.Join(failures, "; "))
	}

	keyPath, err = check.EnsureSSHKey()
	if err != nil {
		return "", cmdSudo, fmt.Errorf("ssh key: %w", err)
	}
	u.Success("SSH key (%s)", keyPath)

	netResult := check.DefaultNetwork(cmdSudo)
	if netResult.Passed {
		u.Success("Default network active")
	} else {
		reason := netResult.Error
		if reason == "" {
			reason = "default network is not active"
		}
		u.Status("%s; starting it", reason)
		if err := check.EnsureDefaultNetwork(cmdSudo); err != nil {
			return "", cmdSudo, fmt.Errorf("default network: %w", err)
		}
		u.Success("Default network ready")
	}

	bridgeName := netName(opts.Network)
	if bridgeName != "default" {
		u.Status("Configuring firewall for bridge %s", bridgeName)
		if err := check.EnsureFirewallRule(bridgeName, cmdSudo); err != nil {
			u.Warn("Firewall rule for bridge %s: %v", bridgeName, err)
		}
	}

	return keyPath, cmdSudo, nil
}

// executeDeploy is the destructive half of deploy. It is called only after the
// interactive review (or directly with --yes).
func executeDeploy(ctx context.Context, u *ui.UI, opts DeployOptions, image detect.ScannedImage, keyPath string) error {
	cmdSudo := hypervisor.NeedsSudo()
	fileSudo := hypervisor.NeedsPrivilegedPath(opts.StoragePath)

	var cleanupFiles []string
	defer func() {
		for _, file := range cleanupFiles {
			sudoRemove(file)
		}
	}()

	if hypervisor.VMExists(opts.Name, cmdSudo) {
		u.Status("Replacing existing VM %q", opts.Name)
		if err := hypervisor.Destroy(opts.Name, cmdSudo); err != nil {
			return fmt.Errorf("replace existing VM: %w", err)
		}
		if hypervisor.VMExists(opts.Name, cmdSudo) {
			return fmt.Errorf("VM %q still exists after virsh undefine", opts.Name)
		}
		for _, path := range []string{
			filepath.Join(opts.StoragePath, opts.Name+".qcow2"),
			filepath.Join(opts.StoragePath, opts.Name+"-seed.iso"),
		} {
			if err := sudoRemove(path); err != nil {
				return err
			}
		}
		if includeDir, err := ssh.IncludeDir(); err == nil {
			if err := ssh.RemoveHostConfig(opts.Name, includeDir); err != nil {
				return err
			}
		}
	}

	hash, err := iso.HashPassword(opts.Password)
	if err != nil {
		return fmt.Errorf("password hash: %w", err)
	}

	cloudData := &iso.CloudInitData{
		Hostname:     opts.Name,
		InstanceID:   fmt.Sprintf("%s-%d", opts.Name, time.Now().Unix()),
		SSHPublicKey: readPubKey(keyPath),
		PasswordHash: hash,
		Username:     os.Getenv("USER"),
		Packages:     opts.Packages,
	}

	seedPath := filepath.Join(opts.StoragePath, opts.Name+"-seed.iso")
	cleanupFiles = append(cleanupFiles, seedPath)

	u.Section(fmt.Sprintf("Creating VM %q", opts.Name))
	if err := u.Spin(ctx, "Cloud-init seed", func(context.Context) error {
		return iso.GenerateSeedISO(cloudData, seedPath, fileSudo)
	}); err != nil {
		return fmt.Errorf("cloud-init iso: %w", err)
	}
	u.Success("Cloud-init seed")

	backingFmt, err := storage.DetectFormat(image.Path, fileSudo)
	if err != nil {
		return fmt.Errorf("detect backing format: %w", err)
	}

	overlayPath := filepath.Join(opts.StoragePath, opts.Name+".qcow2")
	cleanupFiles = append(cleanupFiles, overlayPath)
	if err := u.Spin(ctx, fmt.Sprintf("Disk overlay (%s)", opts.Size), func(context.Context) error {
		return storage.CreateOverlay(image.Path, overlayPath, opts.Size, backingFmt, fileSudo)
	}); err != nil {
		return fmt.Errorf("overlay: %w", err)
	}
	u.Success("Disk overlay (%s)", opts.Size)

	vmCfg := &hypervisor.VMConfig{
		Name:      opts.Name,
		RAM:       opts.RAM,
		CPUs:      opts.CPUs,
		DiskPath:  overlayPath,
		ISOPath:   seedPath,
		Network:   opts.Network,
		OSVariant: image.OS.OSVariant,
	}
	if err := u.Spin(ctx, "VM defined and booted", func(context.Context) error {
		return hypervisor.Deploy(vmCfg, cmdSudo)
	}); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	u.Success("VM defined and booted")

	var ip string
	if err := u.Spin(ctx, "IP address", func(context.Context) error {
		var err error
		ip, err = hypervisor.WaitForIP(opts.Name, netName(opts.Network), cmdSudo, 90*time.Second)
		return err
	}); err != nil {
		return fmt.Errorf("ip discovery: %w\nTry `kdeploy console %s`, inspect `virsh net-dhcp-leases %s`, and check the host firewall", err, opts.Name, netName(opts.Network))
	}
	u.Success("IP address %s", ip)

	exec.Command("ssh-keygen", "-R", ip).Run()

	if err := u.Spin(ctx, "SSH ready", func(context.Context) error {
		return ssh.WaitForSSH(ip, 60*time.Second)
	}); err != nil {
		u.Warn("%v; try `kdeploy console %s`", err, opts.Name)
	} else {
		u.Success("SSH ready")
	}

	includeDir, err := ssh.IncludeDir()
	if err != nil {
		return fmt.Errorf("ssh include dir: %w", err)
	}
	if err := os.MkdirAll(includeDir, 0700); err != nil {
		return fmt.Errorf("ssh include dir: %w", err)
	}
	sshConfig := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	if err := ssh.EnsureInclude(sshConfig); err != nil {
		return fmt.Errorf("ssh config include: %w", err)
	}
	if err := ssh.WriteHostConfig(opts.Name, ip, os.Getenv("USER"), keyPath, includeDir); err != nil {
		return fmt.Errorf("ssh config: %w", err)
	}
	u.Success("SSH config written")

	cleanupFiles = nil

	u.Println("")
	u.Heading("VM ready")
	u.KV("Name", opts.Name, "IP", ip, "SSH", "ssh "+opts.Name)
	u.Println("        ssh " + opts.Name + "-root")
	u.Println("")
	u.Println("Next: kdeploy info " + opts.Name + " · kdeploy console " + opts.Name)
	return nil
}

func readPubKey(keyPath string) string {
	data, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
