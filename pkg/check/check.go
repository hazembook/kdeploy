package check

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hazembook/kdeploy/pkg/storage"
)

type Result struct {
	Name   string
	Passed bool
	Error  string
}

func KVM() Result {
	if _, err := os.Stat("/dev/kvm"); err == nil {
		return Result{Name: "KVM device", Passed: true}
	}
	out, err := exec.Command("lsmod").Output()
	if err == nil && strings.Contains(string(out), "kvm") {
		return Result{Name: "KVM module", Passed: true}
	}
	return Result{Name: "KVM", Passed: false, Error: "KVM not available (enable in BIOS or load kvm module)"}
}

func Libvirt() Result {
	err := exec.Command("virsh", "list").Run()
	if err == nil {
		return Result{Name: "Libvirt", Passed: true}
	}
	err = exec.Command("sudo", "virsh", "list").Run()
	if err == nil {
		return Result{Name: "Libvirt (via sudo)", Passed: true}
	}
	return Result{Name: "Libvirt", Passed: false, Error: "libvirtd not running or not accessible"}
}

func DiskSpace(path string, minKB uint64) Result {
	info, err := storage.CheckSpace(path)
	if err != nil {
		return Result{Name: "Disk space", Passed: false, Error: err.Error()}
	}
	if info.AvailableKB < minKB {
		return Result{
			Name:   "Disk space",
			Passed: false,
			Error:  fmt.Sprintf("only %dMB available at %s, need %dMB", info.AvailableKB/1024, path, minKB/1024),
		}
	}
	return Result{Name: "Disk space", Passed: true}
}

func findSSHKey() (string, bool) {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	}
	for _, key := range candidates {
		pub := key + ".pub"
		if _, err := os.Stat(pub); err == nil {
			return key, true
		}
	}
	return "", false
}

func SSHKey() (string, Result) {
	key, ok := findSSHKey()
	if ok {
		return key, Result{Name: "SSH key", Passed: true}
	}
	home, _ := os.UserHomeDir()
	return "", Result{Name: "SSH key", Passed: false, Error: fmt.Sprintf("no SSH key found in %s/.ssh/", home)}
}

func EnsureSSHKey() (string, error) {
	if key, ok := findSSHKey(); ok {
		return key, nil
	}
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".ssh", "id_ed25519")
	os.MkdirAll(filepath.Join(home, ".ssh"), 0700)
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return keyPath, nil
}

func isNetActive(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Active:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Active:")) == "yes"
		}
	}
	return false
}

func DefaultNetwork(useSudo bool) Result {
	out, err := netInfoOut("default", useSudo)
	if err != nil {
		return Result{Name: "Default network", Passed: false, Error: "default network not defined"}
	}
	if !isNetActive(out) {
		return Result{Name: "Default network", Passed: false, Error: "default network exists but is not active"}
	}
	return Result{Name: "Default network", Passed: true}
}

func netInfoOut(netName string, useSudo bool) (string, error) {
	cmd := exec.Command("virsh", "net-info", netName)
	if useSudo {
		cmd = exec.Command("sudo", "virsh", "net-info", netName)
	}
	out, err := cmd.Output()
	return string(out), err
}

func EnsureDefaultNetwork(useSudo bool) error {
	virsh := func(args ...string) (string, error) {
		cmd := exec.Command("virsh", args...)
		if useSudo {
			cmd = exec.Command("sudo", append([]string{"virsh"}, args...)...)
		}
		out, err := cmd.Output()
		return string(out), err
	}

	_, err := virsh("net-info", "default")
	if err != nil {
		xmlPath := "/usr/share/libvirt/networks/default.xml"
		if _, statErr := os.Stat(xmlPath); statErr != nil {
			return fmt.Errorf("default network XML not found at %s", xmlPath)
		}
		defErr := exec.Command("sudo", "virsh", "net-define", xmlPath).Run()
		if !useSudo {
			defErr = exec.Command("virsh", "net-define", xmlPath).Run()
		}
		if defErr != nil {
			return fmt.Errorf("define default network: %w", defErr)
		}
	}

	out, _ := virsh("net-info", "default")
	if !isNetActive(out) {
		virsh("net-start", "default")
	}

	virsh("net-autostart", "default")
	return nil
}

func EnsureFirewallRule(bridge string, useSudo bool) error {
	if bridge == "" {
		return nil
	}

	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}

	out, err := exec.Command("sudo", "ufw", "status").Output()
	if err != nil || !strings.Contains(string(out), "Status: active") {
		return nil
	}

	rulesOut, _ := exec.Command("sudo", "ufw", "route", "show").Output()
	if strings.Contains(string(rulesOut), bridge) {
		return nil
	}

	cmd := exec.Command("sudo", "ufw", "route", "allow", "in", "on", bridge)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func All(path string, minDiskKB uint64) []Result {
	return []Result{
		KVM(),
		Libvirt(),
		DiskSpace(path, minDiskKB),
	}
}
