package hypervisor

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func GetDomifaddr(name string, useSudo bool) (string, error) {
	args := []string{"virsh", "domifaddr", name, "--source", "agent"}
	if useSudo {
		args = append([]string{"sudo"}, args...)
	}
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "ipv4") {
			continue
		}
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.Contains(f, "/") {
				return strings.Split(f, "/")[0], nil
			}
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
