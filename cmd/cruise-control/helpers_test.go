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
		{"handle root", "root", tc.HandleRoot},
		{"handle 0:1", "0:1", 1},
		{"handle 0:f", "0:ffff", 65535},
		{"handle 1:1", "1:1", 65537},
		{"handle :1", ":1", 1},
		{"handle :ffff", ":ffff", 65535},
		{"handle ffff:", "ffff:", 4294901760},
		{"handle help", "help", 0},
		{"handle interface", "interface", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StrHandle(tt.handle); got != tt.want {
				t.Errorf("failed to parse hanle %s into %d, got %d", tt.handle, tt.want, got)
			}
		})
	}
}
