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
		succes bool
	}{
		{"handle root", "root", tc.HandleRoot, true},
		{"handle 0:1", "0:1", 1, true},
		{"handle 0:f", "0:ffff", 65535, true},
		{"handle 1:1", "1:1", 65537, true},
		{"handle 0:1", "0:1", 1, true},
		{"handle 0:ffff", "0:ffff", 65535, true},
		{"handle ffff:0", "ffff:0", 4294901760, true},
		{"handle help", "help", 0, false},
		{"handle interface", "interface", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := StrHandle(tt.handle); got != tt.want {
				t.Errorf("failed to parse hanle %s into %d, got %d", tt.handle, tt.want, got)
			} else {
				if succes := (err == nil); succes != tt.succes {
					t.Errorf("expected %v here, but got %v\n", tt.succes, succes )
				}
			}
		})
	}
}
