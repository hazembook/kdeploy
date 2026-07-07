package hypervisor

import "testing"

func TestNeedsSudo(t *testing.T) {
	result := NeedsSudo()
	t.Logf("NeedsSudo() = %v", result)
	if result != true && result != false {
		t.Errorf("NeedsSudo() returned non-bool: %v", result)
	}
}

func TestNeedsPrivilegedPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/var/lib/libvirt/images", true},
		{"/var/lib/foo", true},
		{"/srv/vm-images", true},
		{"/opt/vms", true},
		{"/home/user/VM/disks", false},
		{"/tmp/test-vms", false},
		{"/var", false},
		{"/Var/lib/images", false},
	}
	for _, tc := range tests {
		got := NeedsPrivilegedPath(tc.path)
		if got != tc.want {
			t.Errorf("NeedsPrivilegedPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSudoForPath(t *testing.T) {
	if got := SudoForPath("/var/lib/libvirt/images"); len(got) != 1 || got[0] != "sudo" {
		t.Errorf("SudoForPath(/var/...) = %v, want [sudo]", got)
	}
	if got := SudoForPath("/home/user/VM/disks"); got != nil {
		t.Errorf("SudoForPath(/home/...) = %v, want nil", got)
	}
}
