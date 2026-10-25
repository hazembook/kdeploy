package iso

import (
	"fmt"
	"os"
	"os/exec"
)

type CloudInitData struct {
	Hostname     string
	InstanceID   string
	SSHPublicKey string
	PasswordHash string
	Username     string
	Packages     []string
}

func GenerateSeedISO(data *CloudInitData, outputPath string, useSudo bool) error {
	workDir, err := os.MkdirTemp("", "kdeploy-cloudinit-*")
	if err != nil {
		return fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(workDir)

	userData := fmt.Sprintf(`#cloud-config
disable_root: false
users:
  - name: root
    ssh_authorized_keys:
      - %s
  - name: %s
    groups: [ wheel ]
    sudo: [ "ALL=(ALL) NOPASSWD:ALL" ]
    lock_passwd: false
    passwd: %s
    shell: /bin/bash
    ssh_authorized_keys:
      - %s
ssh_pwauth: false
preserve_hostname: false
timezone: UTC
hostname: %s
`, data.SSHPublicKey, data.Username, data.PasswordHash,
		data.SSHPublicKey, data.Hostname)

	if len(data.Packages) > 0 {
		userData += "packages:\n"
		for _, pkg := range data.Packages {
			userData += fmt.Sprintf("  - %s\n", pkg)
		}
	}

	metaData := fmt.Sprintf(`instance-id: %s
local-hostname: %s
`, data.InstanceID, data.Hostname)

	udPath := workDir + "/user-data"
	mdPath := workDir + "/meta-data"
	if err := os.WriteFile(udPath, []byte(userData), 0644); err != nil {
		return fmt.Errorf("write user-data: %w", err)
	}
	if err := os.WriteFile(mdPath, []byte(metaData), 0644); err != nil {
		return fmt.Errorf("write meta-data: %w", err)
	}

	// Use cloud-localds from host (pure-Go ISO generation TBD)
	args := []string{"cloud-localds", outputPath, udPath, mdPath}
	cmd := exec.Command(args[0], args[1:]...)
	if useSudo {
		cmd = exec.Command("sudo", args...)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cloud-localds failed: %w\n%s", err, out)
	}
	return nil
}
