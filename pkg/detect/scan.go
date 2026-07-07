package detect

import (
	"os"
	"path/filepath"
	"sort"
)

type ScannedImage struct {
	Path     string
	Filename string
	OS       ImageInfo
}

func Scan(path string) ([]ScannedImage, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var images []ScannedImage
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".qcow2" && ext != ".img" {
			continue
		}
		fullPath := filepath.Join(path, e.Name())
		info := DetectOS(e.Name())
		images = append(images, ScannedImage{
			Path:     fullPath,
			Filename: e.Name(),
			OS:       info,
		})
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].Filename < images[j].Filename
	})
	return images, nil
}
