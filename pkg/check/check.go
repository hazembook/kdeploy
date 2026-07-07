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

func SSHKey() (string, Result) {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	}
	for _, key := range candidates {
		pub := key + ".pub"
		if _, err := os.Stat(pub); err == nil {
			return key, Result{Name: "SSH key", Passed: true}
		}
	}
	return "", Result{Name: "SSH key", Passed: false, Error: fmt.Sprintf("no SSH key found in %s/.ssh/", home)}
}

func All(path string, minDiskKB uint64) []Result {
	return []Result{
		KVM(),
		Libvirt(),
		DiskSpace(path, minDiskKB),
	}
}
