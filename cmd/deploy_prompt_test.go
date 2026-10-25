package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hazembook/kdeploy/pkg/detect"
)

func TestFirstImage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Rocky-10.qcow2"), []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}

	image, err := firstImage(dir)
	if err != nil {
		t.Fatalf("firstImage: %v", err)
	}
	if image.Filename != "Rocky-10.qcow2" {
		t.Fatalf("filename = %q", image.Filename)
	}
	if image.OS.Distro != "rocky" || image.OS.Release != "10" {
		t.Fatalf("OS detection = %+v", image.OS)
	}
}

func TestFirstImageEmpty(t *testing.T) {
	if _, err := firstImage(t.TempDir()); err == nil {
		t.Fatal("firstImage accepted an empty image directory")
	}
}

func TestParsePackages(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"   ", nil},
		{"none", nil},
		{" NONE ", nil},
		{"nginx", []string{"nginx"}},
		{"nginx python3-pip", []string{"nginx", "python3-pip"}},
	}
	for _, tt := range tests {
		got := parsePackages(tt.input)
		if strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Errorf("parsePackages(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestImageOptions(t *testing.T) {
	images := []detect.ScannedImage{
		{
			Path:     "/images/Rocky-10.qcow2",
			Filename: "Rocky-10.qcow2",
			OS:       detect.ImageInfo{OSVariant: "rocky10", Distro: "rocky", Release: "10"},
		},
	}
	options := imageOptions(images)
	if len(options) != 2 {
		t.Fatalf("options = %d, want 2", len(options))
	}
	if !strings.Contains(options[0].Key, "Rocky-10.qcow2") || !strings.Contains(options[0].Key, "Rocky 10") {
		t.Fatalf("image option = %q", options[0].Key)
	}
	if options[0].Value != images[0].Path {
		t.Fatalf("image option value = %q", options[0].Value)
	}
	if options[1].Value != downloadImageChoice {
		t.Fatalf("download option value = %q", options[1].Value)
	}
}

func TestImageOSLabel(t *testing.T) {
	tests := []struct {
		info detect.ImageInfo
		want string
	}{
		{detect.ImageInfo{Distro: "rocky", Release: "10"}, "Rocky 10"},
		{detect.ImageInfo{Distro: "ubuntu", Release: "24.04"}, "Ubuntu 24.04"},
		{detect.ImageInfo{OSVariant: "custom"}, "custom"},
		{detect.ImageInfo{}, "unknown"},
	}
	for _, tt := range tests {
		if got := imageOSLabel(tt.info); got != tt.want {
			t.Errorf("imageOSLabel(%+v) = %q, want %q", tt.info, got, tt.want)
		}
	}
}

func TestReviewRows(t *testing.T) {
	opts := DeployOptions{
		Name:        "demo",
		RAM:         "2048",
		CPUs:        "2",
		Size:        "20G",
		Network:     "default",
		StoragePath: "/var/lib/libvirt/images",
		Packages:    []string{"nginx"},
	}
	image := detect.ScannedImage{
		Filename: "Rocky-10.qcow2",
		OS:       detect.ImageInfo{Distro: "rocky", Release: "10"},
	}
	rows := strings.Join(reviewRows(opts, image), "|")
	for _, want := range []string{
		"Name|demo",
		"Image|Rocky-10.qcow2",
		"OS|Rocky 10",
		"Resources|2048 MB RAM · 2 vCPU · 20G disk",
		"Network|default",
		"Storage|/var/lib/libvirt/images",
		"SSH|demo, demo-root",
		"Packages|nginx",
	} {
		if !strings.Contains(rows, want) {
			t.Errorf("review rows missing %q: %s", want, rows)
		}
	}
}

func TestReviewActionOptions(t *testing.T) {
	options := reviewActionOptions()
	if len(options) != 4 {
		t.Fatalf("options = %d, want 4", len(options))
	}
	values := make(map[string]bool)
	for _, option := range options {
		values[option.Value] = true
	}
	for _, want := range []string{"create", "edit", "image", "cancel"} {
		if !values[want] {
			t.Errorf("missing review action %q", want)
		}
	}
}

func TestFindImage(t *testing.T) {
	images := []detect.ScannedImage{
		{Path: "/a.qcow2", Filename: "a.qcow2"},
		{Path: "/b.qcow2", Filename: "b.qcow2"},
	}
	if image, ok := findImage(images, "/b.qcow2"); !ok || image.Filename != "b.qcow2" {
		t.Fatalf("findImage returned %+v, %v", image, ok)
	}
	if _, ok := findImage(images, "/missing.qcow2"); ok {
		t.Fatal("findImage found a missing path")
	}
}
