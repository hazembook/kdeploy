package hypervisor

import "testing"

func TestGetIPFromOutput(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{
			input:  `  vnet0     00:11:22:33:44:55  ipv4   192.168.122.100/24  -`,
			expect: "192.168.122.100",
		},
		{
			input:  `  vnet0     00:11:22:33:44:55  ipv4   127.0.0.1/8  -`,
			expect: "",
		},
		{
			input:  `  vnet0     00:11:22:33:44:55  ipv4   10.0.0.5/24  -`,
			expect: "10.0.0.5",
		},
		{
			input:  `no addresses found`,
			expect: "",
		},
	}
	for _, tt := range tests {
		if got := getIPFromOutput(tt.input); got != tt.expect {
			t.Errorf("getIPFromOutput(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestExtractMAC(t *testing.T) {
	xml := `<domain><devices><interface type='network'>
  <mac address='52:54:00:16:01:13'/>
</interface></devices></domain>`
	mac := extractMAC(xml)
	if mac != "52:54:00:16:01:13" {
		t.Errorf("extractMAC: got %q, want 52:54:00:16:01:13", mac)
	}
}

func TestExtractMAC_NotFound(t *testing.T) {
	if mac := extractMAC("<domain></domain>"); mac != "" {
		t.Errorf("expected empty, got %q", mac)
	}
}

func TestExtractIP(t *testing.T) {
	leases := ` 52:54:00:16:01:13 192.168.122.100/24`
	ip := extractIP(leases, "52:54:00:16:01:13")
	if ip != "192.168.122.100" {
		t.Errorf("extractIP: got %q, want 192.168.122.100", ip)
	}
}

func TestExtractIP_NotFound(t *testing.T) {
	if ip := extractIP("no matching leases", "aa:bb:cc:dd:ee:ff"); ip != "" {
		t.Errorf("expected empty, got %q", ip)
	}
}
