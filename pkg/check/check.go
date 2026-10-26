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

var requiredTools = []struct {
	Name    string
	Command string
	Package string // hint for display
}{
	{"virsh", "virsh", "libvirt"},
	{"virt-install", "virt-install", "virt-install"},
	{"cloud-localds", "cloud-localds", "cloud-image-utils / cloud-utils"},
	{"qemu-img", "qemu-img", "qemu-utils / qemu-img"},
	{"openssl", "openssl", "openssl"},
	{"wget/curl", "", "wget curl"},
}

func RequiredTools() []Result {
	var results []Result
	var missing []string

	for _, tool := range requiredTools {
		if tool.Command != "" {
			if _, err := exec.LookPath(tool.Command); err != nil {
				missing = append(missing, tool.Name)
				results = append(results, Result{
					Name: tool.Name, Passed: false,
					Error: fmt.Sprintf("%s not found (install %s)", tool.Command, tool.Package),
				})
			} else {
				results = append(results, Result{Name: tool.Name, Passed: true})
			}
		}
	}

	// wget/curl: need at least one
	if _, errW := exec.LookPath("wget"); errW != nil {
		if _, errC := exec.LookPath("curl"); errC != nil {
			missing = append(missing, "wget/curl")
			results = append(results, Result{
				Name: "wget/curl", Passed: false,
				Error: "neither wget nor curl found (install one)",
			})
		} else {
			results = append(results, Result{Name: "wget/curl", Passed: true})
		}
	} else {
		results = append(results, Result{Name: "wget/curl", Passed: true})
	}

	if len(missing) > 0 {
		results = append(results, Result{
			Name: "Dependencies", Passed: false,
			Error: fmt.Sprintf("missing: %s", strings.Join(missing, ", ")),
		})
	} else {
		results = append(results, Result{Name: "Dependencies", Passed: true})
	}

	return results
}

func DetectDistro() string {
	osRelease, err := os.ReadFile("/etc/os-release")
	if err != nil {
		// Fallback checks
		if _, err := os.Stat("/etc/arch-release"); err == nil {
			return "arch"
		}
		if _, err := os.Stat("/etc/debian_version"); err == nil {
			return "debian"
		}
		if _, err := os.Stat("/etc/fedora-release"); err == nil {
			return "fedora"
		}
		return "unknown"
	}

	id := ""
	idLike := ""
	for _, line := range strings.Split(string(osRelease), "\n") {
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"'")
		}
		if strings.HasPrefix(line, "ID_LIKE=") {
			idLike = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"'")
		}
	}

	distros := []string{id, idLike}
	for _, d := range distros {
		switch d {
		case "arch", "manjaro", "endeavouros", "cachyos":
			return "arch"
		case "ubuntu", "linuxmint", "pop":
			return "ubuntu"
		case "debian":
			return "debian"
		case "fedora", "rhel", "centos":
			return "fedora"
		case "rocky", "almalinux":
			return "rocky"
		}
		if strings.Contains(d, "opensuse") {
			return "opensuse"
		}
	}
	return id
}

func PackageManager(distro string) string {
	switch distro {
	case "arch":
		if _, err := exec.LookPath("pacman"); err == nil {
			return "pacman -S"
		}
	case "ubuntu", "debian":
		if _, err := exec.LookPath("apt"); err == nil {
			return "apt install"
		}
	case "fedora", "rocky":
		if _, err := exec.LookPath("dnf"); err == nil {
			return "dnf install"
		}
	}
	// Direct lookup by binary
	if _, err := exec.LookPath("pacman"); err == nil {
		return "pacman -S"
	}
	if _, err := exec.LookPath("apt"); err == nil {
		return "apt install"
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		return "dnf install"
	}
	if _, err := exec.LookPath("yum"); err == nil {
		return "yum install"
	}
	if _, err := exec.LookPath("zypper"); err == nil {
		return "zypper install"
	}
	return "unknown"
}

var distroPackages = map[string]string{
	"arch":     "libvirt virt-install qemu-img cloud-utils openssl wget curl",
	"debian":   "libvirt-daemon-system virtinst qemu-utils cloud-image-utils openssl wget curl",
	"ubuntu":   "libvirt-daemon-system virtinst qemu-utils cloud-image-utils openssl wget curl",
	"fedora":   "libvirt virt-install qemu-img cloud-utils openssl wget curl",
	"rocky":    "libvirt virt-install qemu-img cloud-utils openssl wget curl",
	"opensuse": "libvirt virt-install qemu-tools cloud-utils openssl wget curl",
}

func DistroPackageList(distro string) string {
	if pkgs, ok := distroPackages[distro]; ok {
		return pkgs
	}
	return ""
}

func InstallDependencies(distro, pkgManager string) error {
	pkgs := DistroPackageList(distro)
	if pkgs == "" {
		return fmt.Errorf("no package list for distro %q", distro)
	}

	if !strings.Contains(pkgManager, " ") {
		return fmt.Errorf("unknown package manager format: %s", pkgManager)
	}

	parts := strings.SplitN(pkgManager, " ", 2)
	pkgMgr := parts[0]
	installArg := parts[1]

	var cmd *exec.Cmd
	switch pkgMgr {
	case "pacman":
		cmd = exec.Command("sudo", pkgMgr, "-S", "--needed")
		cmd.Args = append(cmd.Args, strings.Fields(pkgs)...)
	case "apt":
		cmd = exec.Command("sudo", "apt", "update")
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("apt update: %w", err)
		}
		cmd = exec.Command("sudo", pkgMgr, installArg, "-y")
		cmd.Args = append(cmd.Args, strings.Fields(pkgs)...)
	default:
		cmd = exec.Command("sudo", pkgMgr, installArg, "-y")
		cmd.Args = append(cmd.Args, strings.Fields(pkgs)...)
	}

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	return nil
}

func All(path string, minDiskKB uint64) []Result {
	return []Result{
		KVM(),
		Libvirt(),
		DiskSpace(path, minDiskKB),
	}
}
