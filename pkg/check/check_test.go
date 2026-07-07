package check

import (
	"os"
	"strings"
	"testing"
)

func TestKVM(t *testing.T) {
	r := KVM()
	t.Logf("KVM: passed=%v err=%q", r.Passed, r.Error)
}

func TestSSHKey(t *testing.T) {
	path, r := SSHKey()
	t.Logf("SSHKey: passed=%v path=%q err=%q", r.Passed, path, r.Error)
}

func TestDiskSpace(t *testing.T) {
	dir := t.TempDir()
	r := DiskSpace(dir, 1024)
	if !r.Passed {
		t.Errorf("expected disk check to pass in temp dir: %s", r.Error)
	}
}

func TestDiskSpace_NoSpace(t *testing.T) {
	r := DiskSpace("/nonexistent", 1024)
	if r.Passed {
		t.Fatal("expected failure for nonexistent path")
	}
}

func TestEnsureSSHKey(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", oldHome)

	keyPath, err := EnsureSSHKey()
	if err != nil {
		t.Fatalf("EnsureSSHKey: %v", err)
	}
	if !strings.Contains(keyPath, "id_ed25519") {
		t.Errorf("expected ed25519 key, got %q", keyPath)
	}
	if _, err := os.Stat(keyPath + ".pub"); err != nil {
		t.Errorf("public key not found: %v", err)
	}
}

func TestDetectDistro(t *testing.T) {
	distro := DetectDistro()
	if distro == "" {
		t.Fatal("expected non-empty distro")
	}
	t.Logf("Detected distro: %s", distro)
}

func TestDistroPackageList(t *testing.T) {
	for _, d := range []string{"arch", "debian", "ubuntu", "fedora", "rocky", "opensuse"} {
		pkgs := DistroPackageList(d)
		if pkgs == "" {
			t.Errorf("expected packages for distro %q", d)
		}
	}
	if pkgs := DistroPackageList("unknown"); pkgs != "" {
		t.Errorf("expected empty for unknown distro, got %q", pkgs)
	}
}

func TestRequiredTools(t *testing.T) {
	results := RequiredTools()
	// We expect at least some tools to exist on the test system
	found := false
	for _, r := range results {
		if r.Passed {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one tool to be found")
	}
}

func TestIsNetActive(t *testing.T) {
	tests := []struct {
		input  string
		expect bool
	}{
		{"Active:          yes", true},
		{"Active:          no", false},
		{"  Name: default\nActive:          yes", true},
		{"  Name: default\nState: inactive", false},
	}
	for _, tt := range tests {
		if got := isNetActive(tt.input); got != tt.expect {
			t.Errorf("isNetActive(%q) = %v, want %v", tt.input, got, tt.expect)
		}
	}
}

func TestAll(t *testing.T) {
	dir := t.TempDir()
	results := All(dir, 1024)
	if len(results) == 0 {
		t.Fatal("expected at least one check")
	}
}

func TestLibvirt(t *testing.T) {
	r := Libvirt()
	t.Logf("Libvirt: passed=%v err=%q", r.Passed, r.Error)
}
