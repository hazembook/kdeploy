package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"Rocky-10-GenericCloud-Base.latest.x86_64.qcow2",
		"noble-server-cloudimg-amd64.img",
		"debian-12-generic-amd64.qcow2",
		"readme.txt",
		"image.tar.gz",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	images, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != 3 {
		t.Fatalf("expected 3 images, got %d", len(images))
	}

	expected := map[string]string{
		"Rocky-10-GenericCloud-Base.latest.x86_64.qcow2": "rocky10",
		"noble-server-cloudimg-amd64.img":                "ubuntu24.04",
		"debian-12-generic-amd64.qcow2":                  "debian12",
	}
	for _, img := range images {
		want, ok := expected[img.Filename]
		if !ok {
			t.Errorf("unexpected image: %s", img.Filename)
			continue
		}
		if img.OS.OSVariant != want {
			t.Errorf("%s: got %q, want %q", img.Filename, img.OS.OSVariant, want)
		}
	}
}

func TestScan_NotFound(t *testing.T) {
	_, err := Scan("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}
