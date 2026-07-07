package storage

import "strings"

func splitLines(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func trimAfter(s, sep string) string {
	if i := strings.Index(s, sep); i >= 0 {
		return strings.TrimSpace(s[i+len(sep):])
	}
	return strings.TrimSpace(s)
}
