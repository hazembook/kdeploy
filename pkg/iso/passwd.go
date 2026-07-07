package iso

import (
	"fmt"
	"os/exec"
	"strings"
)

func HashPassword(password string) (string, error) {
	out, err := exec.Command("openssl", "passwd", "-6", password).Output()
	if err != nil {
		return "", fmt.Errorf("openssl passwd failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
