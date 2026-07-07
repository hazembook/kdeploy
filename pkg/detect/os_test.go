package detect

import "testing"

func TestDetectOS(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"Rocky-10-GenericCloud-Base.latest.x86_64.qcow2", "rocky10"},
		{"Rocky-9-GenericCloud-Base.latest.x86_64.qcow2", "rocky9"},
		{"noble-server-cloudimg-amd64.img", "ubuntu24.04"},
		{"jammy-server-cloudimg-amd64.img", "ubuntu22.04"},
		{"debian-12-generic-amd64.qcow2", "debian12"},
		{"generic_alpine-3.23.3-x86_64-bios-cloudinit-r0.qcow2", "alpinelinux3.23"},
		{"Fedora-Cloud-Base-41-1.4.x86_64.qcow2", "fedora41"},
		{"unknown-image.qcow2", "generic"},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := DetectOS(tt.filename)
			if got.OSVariant != tt.want {
				t.Errorf("DetectOS(%q) = %q, want %q", tt.filename, got.OSVariant, tt.want)
			}
		})
	}
}
