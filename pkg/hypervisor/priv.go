package hypervisor

import (
	"os/exec"
	"strings"
)

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
