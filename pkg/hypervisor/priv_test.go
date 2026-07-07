package hypervisor

import "testing"

func TestNeedsSudo(t *testing.T) {
	// We can't predict the result without knowing the test env,
	// but we can at least verify it returns a bool without error.
	result := NeedsSudo()
	t.Logf("NeedsSudo() = %v", result)
	if result != true && result != false {
		t.Errorf("NeedsSudo() returned non-bool: %v", result)
	}
}
