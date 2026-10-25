package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	vmNamePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	diskSizePattern = regexp.MustCompile(`^[1-9][0-9]*[GgMmTt]$`)
	networkPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	macPattern      = regexp.MustCompile(`^[0-9A-Fa-f]{2}(:[0-9A-Fa-f]{2}){5}$`)
	packagePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+._:-]*$`)
)

// ValidateVMName rejects names that libvirt cannot represent consistently.
func ValidateVMName(name string) error {
	if !vmNamePattern.MatchString(name) {
		return fmt.Errorf("invalid VM name %q: use 1-64 letters, digits, dots, underscores, or hyphens, starting with a letter or digit", name)
	}
	return nil
}

// ValidateDiskSize requires an explicit G, M, or T suffix so qemu-img never
// receives an ambiguous bare byte count.
func ValidateDiskSize(size string) error {
	if !diskSizePattern.MatchString(size) {
		return fmt.Errorf("invalid disk size %q: use a positive number with G, M, or T (for example 20G)", size)
	}
	return nil
}

// ValidateRAM validates a positive memory value in megabytes.
func ValidateRAM(ram string) error {
	return validatePositiveInt("RAM", ram)
}

// ValidateCPUs validates a positive vCPU count.
func ValidateCPUs(cpus string) error {
	return validatePositiveInt("vCPUs", cpus)
}

func validatePositiveInt(label, value string) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		return fmt.Errorf("invalid %s %q: use a positive integer", label, value)
	}
	return nil
}

// ValidateNetwork accepts a libvirt network name or name,mac=MAC.
func ValidateNetwork(network string) error {
	parts := strings.SplitN(network, ",", 2)
	if len(parts) == 0 || !networkPattern.MatchString(parts[0]) {
		return fmt.Errorf("invalid network %q: use a network name, optionally followed by ,mac=AA:BB:CC:DD:EE:FF", network)
	}
	if len(parts) == 1 {
		return nil
	}
	suffix := parts[1]
	if !strings.HasPrefix(suffix, "mac=") {
		return fmt.Errorf("invalid network %q: the only supported suffix is mac=AA:BB:CC:DD:EE:FF", network)
	}
	mac := strings.TrimPrefix(suffix, "mac=")
	if !macPattern.MatchString(mac) {
		return fmt.Errorf("invalid MAC address %q", mac)
	}
	return nil
}

// ValidatePackages rejects package tokens that could be interpreted as flags
// or shell fragments by the cloud-init installer.
func ValidatePackages(packages []string) error {
	for _, pkg := range packages {
		if !packagePattern.MatchString(pkg) {
			return fmt.Errorf("invalid package name %q", pkg)
		}
	}
	return nil
}
