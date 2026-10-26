package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hazembook/kdeploy/pkg/ssh"
)

func TestVMResourcePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	storage := t.TempDir()

	disk, seed, sshConfig, err := vmResourcePaths(storage, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if disk != filepath.Join(storage, "demo.qcow2") {
		t.Fatalf("disk = %q", disk)
	}
	if seed != filepath.Join(storage, "demo-seed.iso") {
		t.Fatalf("seed = %q", seed)
	}
	includeDir, _ := ssh.IncludeDir()
	if sshConfig != filepath.Join(includeDir, "demo.conf") {
		t.Fatalf("ssh config = %q", sshConfig)
	}
}

func TestRemoveVMResources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	storage := t.TempDir()
	includeDir, err := ssh.IncludeDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(includeDir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(storage, "demo.qcow2"),
		filepath.Join(storage, "demo-seed.iso"),
		filepath.Join(includeDir, "demo.conf"),
	} {
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	if err := removeVMResources("demo", storage); err != nil {
		t.Fatalf("removeVMResources: %v", err)
	}
	for _, path := range []string{
		filepath.Join(storage, "demo.qcow2"),
		filepath.Join(storage, "demo-seed.iso"),
		filepath.Join(includeDir, "demo.conf"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s still exists: %v", path, err)
		}
	}
}

func TestExistingResources(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "present")
	if err := os.WriteFile(present, nil, 0600); err != nil {
		t.Fatal(err)
	}
	found, err := existingResources(present, filepath.Join(dir, "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0] != present {
		t.Fatalf("existingResources = %v", found)
	}
}
