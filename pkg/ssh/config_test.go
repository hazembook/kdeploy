package ssh

import (
	"testing"
	"time"
)

func TestWaitForSSH_Timeout(t *testing.T) {
	err := WaitForSSH("198.51.100.1", 2*time.Second)
	if err == nil {
		t.Fatal("expected timeout for unroutable IP")
	}
	t.Logf("Got expected error: %v", err)
}
