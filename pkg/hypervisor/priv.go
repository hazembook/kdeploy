package hypervisor

import (
	"os/exec"
	"strings"
)

// NeedsSudo returns true when the user is NOT in the libvirt group,
// meaning virsh/virt-install need sudo/doas.
func NeedsSudo() bool {
	out, err := exec.Command("id", "-nG").Output()
	if err != nil {
		return true
	}
	for _, group := range strings.Fields(string(out)) {
		if group == "libvirt" {
			return false
		}
	}
	return true
}

// NeedsPrivilegedPath returns true when path is under a system directory
// (/var/, /srv/, /opt/) and needs sudo for file operations regardless
// of libvirt group membership.
func NeedsPrivilegedPath(path string) bool {
	return strings.HasPrefix(path, "/var/") ||
		strings.HasPrefix(path, "/srv/") ||
		strings.HasPrefix(path, "/opt/")
}

// SudoForPath returns ["sudo"] when NeedsPrivilegedPath(path), else nil.
// Use it to prefix file-operation commands like qemu-img, cloud-localds, rm.
func SudoForPath(path string) []string {
	if NeedsPrivilegedPath(path) {
		return []string{"sudo"}
	}
	return nil
}
