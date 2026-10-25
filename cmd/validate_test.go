package cmd

import "testing"

func TestValidateVMName(t *testing.T) {
	for _, name := range []string{"demo", "demo-1", "demo_1", "Demo.1"} {
		if err := ValidateVMName(name); err != nil {
			t.Errorf("ValidateVMName(%q) = %v", name, err)
		}
	}
	for _, name := range []string{"", "-demo", "demo name", "demo/1"} {
		if err := ValidateVMName(name); err == nil {
			t.Errorf("ValidateVMName(%q) accepted", name)
		}
	}
}

func TestValidateDiskSize(t *testing.T) {
	for _, size := range []string{"20G", "1M", "2T", "20g"} {
		if err := ValidateDiskSize(size); err != nil {
			t.Errorf("ValidateDiskSize(%q) = %v", size, err)
		}
	}
	for _, size := range []string{"", "0G", "20", "20GB", "-1G", "G"} {
		if err := ValidateDiskSize(size); err == nil {
			t.Errorf("ValidateDiskSize(%q) accepted", size)
		}
	}
}

func TestValidateResources(t *testing.T) {
	for _, value := range []string{"1", "2048", " 4096 "} {
		if err := ValidateRAM(value); err != nil {
			t.Errorf("ValidateRAM(%q) = %v", value, err)
		}
		if err := ValidateCPUs(value); err != nil {
			t.Errorf("ValidateCPUs(%q) = %v", value, err)
		}
	}
	for _, value := range []string{"", "0", "-1", "two"} {
		if err := ValidateRAM(value); err == nil {
			t.Errorf("ValidateRAM(%q) accepted", value)
		}
		if err := ValidateCPUs(value); err == nil {
			t.Errorf("ValidateCPUs(%q) accepted", value)
		}
	}
}

func TestValidateNetwork(t *testing.T) {
	for _, network := range []string{"default", "br0", "net_1", "default,mac=52:54:00:12:34:56"} {
		if err := ValidateNetwork(network); err != nil {
			t.Errorf("ValidateNetwork(%q) = %v", network, err)
		}
	}
	for _, network := range []string{"", "bad network", "-br0", "default,foo=bar", "default,mac=bad"} {
		if err := ValidateNetwork(network); err == nil {
			t.Errorf("ValidateNetwork(%q) accepted", network)
		}
	}
}

func TestValidatePackages(t *testing.T) {
	if err := ValidatePackages([]string{"nginx", "python3-pip", "golang-1.22"}); err != nil {
		t.Errorf("valid packages rejected: %v", err)
	}
	for _, pkg := range []string{"", "--force", "bad pkg", "pkg;rm"} {
		if err := ValidatePackages([]string{pkg}); err == nil {
			t.Errorf("ValidatePackages(%q) accepted", pkg)
		}
	}
}
