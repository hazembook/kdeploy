package check

import (
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
	r := DiskSpace(dir, 1024) // need 1MB
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

func TestSSHKey_NotFound(t *testing.T) {
	_, r := SSHKey()
	t.Logf("SSHKey result: passed=%v err=%q", r.Passed, r.Error)
}
