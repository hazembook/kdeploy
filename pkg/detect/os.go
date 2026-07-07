package detect

import "strings"

type ImageInfo struct {
	OSVariant string
	Distro    string
	Release   string
}

func DetectOS(filename string) ImageInfo {
	lower := strings.ToLower(filename)

	switch {
	case contains(lower, "rocky", "10"):
		return ImageInfo{OSVariant: "rocky10", Distro: "rocky", Release: "10"}
	case contains(lower, "rocky", "9"):
		return ImageInfo{OSVariant: "rocky9", Distro: "rocky", Release: "9"}
	case contains(lower, "rocky", "8"):
		return ImageInfo{OSVariant: "rocky8", Distro: "rocky", Release: "8"}
	case contains(lower, "alma", "10"):
		return ImageInfo{OSVariant: "almalinux10", Distro: "almalinux", Release: "10"}
	case contains(lower, "alma", "9"):
		return ImageInfo{OSVariant: "almalinux9", Distro: "almalinux", Release: "9"}
	case contains(lower, "alma", "8"):
		return ImageInfo{OSVariant: "almalinux8", Distro: "almalinux", Release: "8"}
	case contains(lower, "noble") || contains(lower, "24.04") || strings.HasSuffix(lower, "2404"):
		return ImageInfo{OSVariant: "ubuntu24.04", Distro: "ubuntu", Release: "24.04"}
	case contains(lower, "jammy") || contains(lower, "22.04") || strings.HasSuffix(lower, "2204"):
		return ImageInfo{OSVariant: "ubuntu22.04", Distro: "ubuntu", Release: "22.04"}
	case contains(lower, "focal") || contains(lower, "20.04") || strings.HasSuffix(lower, "2004"):
		return ImageInfo{OSVariant: "ubuntu20.04", Distro: "ubuntu", Release: "20.04"}
	case contains(lower, "debian", "13"):
		return ImageInfo{OSVariant: "debian13", Distro: "debian", Release: "13"}
	case contains(lower, "debian", "12"):
		return ImageInfo{OSVariant: "debian12", Distro: "debian", Release: "12"}
	case contains(lower, "debian", "11"):
		return ImageInfo{OSVariant: "debian11", Distro: "debian", Release: "11"}
	case contains(lower, "alpine", "3.23"):
		return ImageInfo{OSVariant: "alpinelinux3.23", Distro: "alpine", Release: "3.23"}
	case contains(lower, "alpine", "3.22"):
		return ImageInfo{OSVariant: "alpinelinux3.22", Distro: "alpine", Release: "3.22"}
	case contains(lower, "fedora", "41"):
		return ImageInfo{OSVariant: "fedora41", Distro: "fedora", Release: "41"}
	case contains(lower, "fedora", "40"):
		return ImageInfo{OSVariant: "fedora40", Distro: "fedora", Release: "40"}
	default:
		return ImageInfo{OSVariant: "generic", Distro: "unknown", Release: "unknown"}
	}
}

func contains(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
