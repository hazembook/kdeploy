package hypervisor

import (
	"testing"
	"time"
)

func TestWaitForIP_NoVM(t *testing.T) {
	// Should timeout quickly for a non-existent VM
	_, err := WaitForIP("nonexistent-vm-12345", "default", NeedsSudo(), 2*time.Second)
	if err == nil {
		t.Fatal("expected timeout error for nonexistent VM")
	}
	t.Logf("Got expected error: %v", err)
}
