package hypervisor

import (
	"regexp"
	"strings"
)

var macRe = regexp.MustCompile(`mac address='([^']+)'`)

func extractMAC(xml string) string {
	m := macRe.FindStringSubmatch(xml)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractIP(leases string, mac string) string {
	mac = strings.ToLower(mac)
	for _, line := range strings.Split(leases, "\n") {
		if strings.Contains(strings.ToLower(line), mac) {
			fields := strings.Fields(line)
			for _, f := range fields {
				if strings.Contains(f, "/") {
					return strings.Split(f, "/")[0]
				}
			}
		}
	}
	return ""
}
