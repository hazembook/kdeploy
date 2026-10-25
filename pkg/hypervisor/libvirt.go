package hypervisor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type VMConfig struct {
	Name      string
	RAM       string
	CPUs      string
	DiskPath  string
	ISOPath   string
	Network   string
	OSVariant string
}

func Deploy(cfg *VMConfig, useSudo bool) error {
	args := []string{
		"virt-install",
		"--name", cfg.Name,
		"--memory", cfg.RAM,
		"--vcpus", cfg.CPUs,
		"--disk", fmt.Sprintf("path=%s,format=qcow2,bus=virtio", cfg.DiskPath),
		"--disk", fmt.Sprintf("path=%s,device=cdrom", cfg.ISOPath),
		"--os-variant", cfg.OSVariant,
		"--graphics", "vnc,listen=127.0.0.1",
		"--console", "pty,target_type=serial",
		"--network", fmt.Sprintf("network=%s", cfg.Network),
		"--import",
		"--noautoconsole",
	}

	cmd := exec.Command("sudo", args...)
	if !useSudo {
		cmd = exec.Command(args[0], args[1:]...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("virt-install failed: %w\n%s", err, out)
	}
	return nil
}

func Destroy(name string, useSudo bool) error {
	prefix := []string{}
	if useSudo {
		prefix = []string{"sudo"}
	}

	cmds := [][]string{
		append(prefix, "virsh", "destroy", name),
		append(prefix, "virsh", "undefine", name, "--remove-all-storage"),
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		_ = cmd.Run() // best-effort
	}
	return nil
}

type VMEntry struct {
	ID    string
	Name  string
	State string
}

func List(useSudo bool) ([]VMEntry, error) {
	args := []string{"virsh", "list", "--all"}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return nil, fmt.Errorf("virsh list: %w", err)
	}
	var entries []VMEntry
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, " Id") {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			entries = append(entries, VMEntry{
				ID:    fields[0],
				Name:  fields[1],
				State: strings.Join(fields[2:], " "),
			})
		}
	}
	return entries, nil
}

func VMStart(name string, useSudo bool) error {
	args := []string{"virsh", "start", name}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("virsh start: %w\n%s", err, out)
	}
	return nil
}

func VMShutdown(name string, useSudo bool) error {
	args := []string{"virsh", "shutdown", name}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("virsh shutdown: %w\n%s", err, out)
	}
	return nil
}

func VMStop(name string, useSudo bool) error {
	args := []string{"virsh", "destroy", name}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("virsh destroy: %w\n%s", err, out)
	}
	return nil
}

func VMReboot(name string, useSudo bool) error {
	args := []string{"virsh", "reboot", name}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("virsh reboot: %w\n%s", err, out)
	}
	return nil
}

func VMConsole(name string, useSudo bool) error {
	args := []string{"virsh", "console", name}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func VMEdit(name string, useSudo bool) error {
	args := []string{"virsh", "edit", name}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func VMExists(name string, useSudo bool) bool {
	args := []string{"virsh", "dominfo", name}
	cmd := exec.Command(args[0], args[1:]...)
	if useSudo {
		cmd = exec.Command("sudo", args...)
	}
	return cmd.Run() == nil
}

func GetMAC(name string, useSudo bool) (string, error) {
	args := []string{"virsh", "dumpxml", name}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return "", fmt.Errorf("dumpxml: %w", err)
	}
	return extractMAC(string(out)), nil
}

func GetDHCPLease(network string, mac string, useSudo bool) (string, error) {
	args := []string{"virsh", "net-dhcp-leases", network}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return "", fmt.Errorf("net-dhcp-leases: %w", err)
	}
	return extractIP(string(out), mac), nil
}
