package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

const includeDirName = "kdeploy.d"

func IncludeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".ssh", includeDirName), nil
}

func EnsureInclude(configPath string) error {
	includeLine := fmt.Sprintf("Include %s/*", includeDirName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(configPath, []byte(includeLine+"\n"), 0644)
		}
		return fmt.Errorf("read ssh config: %w", err)
	}

	if !contains(string(data), includeDirName) {
		newData := includeLine + "\n" + string(data)
		return os.WriteFile(configPath, []byte(newData), 0644)
	}
	return nil
}

func WriteHostConfig(name, ip, user, keyPath, includeDir string) error {
	content := fmt.Sprintf(`Host %s %s-%s
    HostName %s
    User %s
    IdentityFile %s
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null

Host %s-root
    HostName %s
    User root
    IdentityFile %s
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
`, name, name, user, ip, user, keyPath, name, ip, keyPath)

	return os.WriteFile(filepath.Join(includeDir, name+".conf"), []byte(content), 0600)
}

func RemoveHostConfig(name, includeDir string) error {
	path := filepath.Join(includeDir, name+".conf")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove ssh config: %w", err)
	}
	return nil
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func WaitForSSH(ip string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "22"), 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("SSH not reachable at %s:22 after %v", ip, timeout)
}
