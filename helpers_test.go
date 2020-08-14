package main

import (
	"testing"

	"github.com/florianl/go-tc"
)

func TestStrHandle(t *testing.T) {
	tests := []struct {
		name   string
		handle string
		want   uint32
	}{
		{"root handle", "root", tc.HandleRoot},
		{"root 0:1", "0:1", 1},
		{"root 0:f", "0:ffff", 65535},
		{"root 1:1", "1:1", 65537},
		{"root :1", ":1", 1},
		{"root :ffff", ":ffff", 65535},
		{"root ffff:", "ffff:", 4294901760},
		{"root 1:", "1:", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StrHandle(tt.handle); got != tt.want {
				t.Errorf("failed to parse hanle %s into %d, got %d", tt.handle, tt.want, got)
			}
		})
	}
}
