package cmd

import (
	"strings"
	"testing"

	"github.com/hazembook/kdeploy/pkg/check"
)

func TestCheckExit(t *testing.T) {
	tests := []struct {
		name    string
		results []check.Result
		wantErr bool
		want    string
	}{
		{
			name:    "all pass",
			results: []check.Result{{Name: "virsh", Passed: true}, {Name: "qemu-img", Passed: true}},
		},
		{
			name:    "one failure",
			results: []check.Result{{Name: "virsh", Passed: true}, {Name: "qemu-img", Passed: false, Error: "not found"}},
			wantErr: true,
			want:    "1/2 checks passed",
		},
		{
			name:    "failure detail included",
			results: []check.Result{{Name: "cloud-localds", Passed: false, Error: "cloud-localds not found"}},
			wantErr: true,
			want:    "cloud-localds not found",
		},
		{
			name:    "empty failure uses fallback",
			results: []check.Result{{Name: "openssl", Passed: false}},
			wantErr: true,
			want:    "openssl: failed",
		},
		{
			name:    "empty results",
			results: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkExit(tt.results)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkExit error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("checkExit error = %q, want substring %q", err, tt.want)
			}
		})
	}
}
