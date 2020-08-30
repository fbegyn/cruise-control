package main

import (
	"testing"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

func TestNewNode(t *testing.T) {
	tests := []struct {
		testName string
		name     string
		typ      string
		expected *Node
	}{
		{"TestEmptyClass", "class_test1", "class", &Node{
			Name:     "class_test1",
			Type:     "class",
			Children: []*Node{},
		}},
		{"TestEmptyQdisc", "qdisc_test1", "qdisc", &Node{
			Name:     "qdisc_test1",
			Type:     "qdisc",
			Children: []*Node{},
		}},
		{"TestEmptyFilter", "filter_test1", "filter", &Node{
			Name:     "filter_test1",
			Type:     "filter",
			Children: []*Node{},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			result := NewNode(tt.name, tt.typ)
			if !result.equalHeader(tt.expected) {
				t.Errorf("Failed to create node with correct header. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result, tt.expected,
				)
			}
			if !result.equalObject(tt.expected) {
				t.Errorf("Failed to create node with correct object. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result, tt.expected,
				)
			}
		})
	}
}

func TestNewNodeWithObject(t *testing.T) {

	testQdisc := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: 10,
			Handle:  10,
			Parent:  tc.HandleRoot,
			Info:    0,
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			HfscQOpt: &tc.HfscQOpt{
				DefCls: 12,
			},
		},
	}
	testQdisc2 := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: 10,
			Handle:  10,
			Parent:  tc.HandleRoot,
			Info:    0,
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			HfscQOpt: &tc.HfscQOpt{
				DefCls: 12,
			},
		},
	}

	tests := []struct {
		testName string
		name     string
		typ      string
		object   tc.Object
		expected *Node
	}{
		{"TestQdisc", "qdisc_test1", "qdisc", testQdisc, &Node{
			Name:     "qdisc_test1",
			Type:     "qdisc",
			Object:   testQdisc2,
			Children: []*Node{},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			result := NewNodeWithObject(tt.name, tt.typ, tt.object)
			if !result.equalHeader(tt.expected) {
				t.Errorf("Failed to create node with correct header. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result, tt.expected,
				)
			}
			if !result.equalObject(tt.expected) {
				t.Errorf("Failed to create node with correct object. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result.Object, tt.expected.Object,
				)
			}
		})
	}
}
