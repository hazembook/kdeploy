package hypervisor

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func getIPFromOutput(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "ipv4") {
			continue
		}
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.Contains(f, "/") {
				ip := strings.Split(f, "/")[0]
				if !strings.HasPrefix(ip, "127.") {
					return ip
				}
			}
		}
	}
	return ""
}

func GetDomifaddr(name string, useSudo bool) (string, error) {
	// Try agent source first (requires qemu-guest-agent in VM)
	args := []string{"virsh", "domifaddr", name, "--source", "agent"}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err == nil {
		if ip := getIPFromOutput(string(out)); ip != "" {
			return ip, nil
		}
	}

	// Fallback to ARP/lease source (always available)
	args = []string{"virsh", "domifaddr", name}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err = exec.Command(args[0], args[1:]...).Output()
	if err == nil {
		if ip := getIPFromOutput(string(out)); ip != "" {
			return ip, nil
		}
	}

	return "", fmt.Errorf("no ipv4 address found")
}

func WaitForIP(name, network string, useSudo bool, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	mac, err := GetMAC(name, useSudo)
	if err != nil {
		return "", fmt.Errorf("get mac: %w", err)
	}

	for time.Now().Before(deadline) {
		ip, err := GetDomifaddr(name, useSudo)
		if err == nil && ip != "" {
			return ip, nil
		}

		ip, err = GetDHCPLease(network, mac, useSudo)
		if err == nil && ip != "" {
			return ip, nil
		}

		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("timed out waiting for IP after %v", timeout)
}
