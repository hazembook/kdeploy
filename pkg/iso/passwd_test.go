package iso

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("linux")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$6$") {
		t.Errorf("expected SHA-512 crypt ($6$) prefix, got: %s", hash)
	}
	if len(hash) < 80 {
		t.Errorf("hash too short (%d chars): %s", len(hash), hash)
	}
	t.Logf("Hash: %s", hash)
}
