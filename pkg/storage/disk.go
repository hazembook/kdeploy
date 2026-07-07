package storage

import (
	"fmt"
	"os/exec"
	"syscall"
)

type DiskInfo struct {
	AvailableKB uint64
	TotalKB     uint64
}

func CheckSpace(path string) (*DiskInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("statfs %s: %w", path, err)
	}
	return &DiskInfo{
		AvailableKB: stat.Bavail * uint64(stat.Bsize) / 1024,
		TotalKB:     stat.Blocks * uint64(stat.Bsize) / 1024,
	}, nil
}

func CreateOverlay(backingPath, overlayPath, size, backingFmt string) error {
	args := []string{
		"qemu-img", "create", "-f", "qcow2",
		"-b", backingPath, "-F", backingFmt,
		overlayPath, size,
	}
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img create overlay: %w\n%s", err, out)
	}
	return nil
}

func DetectFormat(path string) (string, error) {
	out, err := exec.Command("qemu-img", "info", path).Output()
	if err != nil {
		return "", fmt.Errorf("qemu-img info: %w", err)
	}
	return parseFormat(string(out)), nil
}

func parseFormat(info string) string {
	for _, line := range splitLines(info) {
		if contains(line, "file format") {
			return trimAfter(trimAfter(line, ":"), " ")
		}
	}
	return "qcow2"
}
