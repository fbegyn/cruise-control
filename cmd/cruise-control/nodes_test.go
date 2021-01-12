package main

import (
	"testing"
	"reflect"

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
			if !result.equalNode(*tt.expected) {
				t.Errorf("Failed to create node with correct header. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result, tt.expected,
				)
			}
			if !result.equalMsg(*tt.expected) {
				t.Errorf("Failed to create an object with the same Msg. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result.Object, tt.expected.Object,
				)
			}
			if !result.equalKind(*tt.expected) {
				t.Errorf("Failed to create an object with the same kind. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result.Object, tt.expected.Object,
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
			if !result.equalNode(*tt.expected) {
				t.Errorf("Failed to create node with correct header. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result, tt.expected,
				)
			}
			if !result.equalMsg(*tt.expected) {
				t.Errorf("Failed to create an object with the same Msg. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result.Object, tt.expected.Object,
				)
			}
			if !result.equalKind(*tt.expected) {
				t.Errorf("Failed to create an object with the same kind. name: %s and type: %s\nGot: %v\nExpected: %v\n",
					tt.name, tt.typ, result.Object, tt.expected.Object,
				)
			}
		})
	}
}

func TestAddChild(t *testing.T){
	testNode := NewNode("parent", "testing")
	testChild := NewNode("child", "testing")
	t.Run("addChild", func(t *testing.T){
		testNode.addChild(testChild)
		if len(testNode.Children) != 1 {
			t.Errorf("failed to add child to node")
		}
		for _, ch := range testNode.Children {
			if equal := reflect.DeepEqual(ch, testChild); equal {
				break
			} else {
				t.Errorf("no child of %v the added child %v\n", testNode, testChild)
			}
		}
	})
}

func TestToParent(t *testing.T){
	testNode := NewNode("parent", "testing")
	testChild := NewNode("child", "testing")
	t.Run("addToParent", func(t *testing.T){
		testChild.addToParent(testNode)
		if len(testNode.Children) != 1 {
			t.Errorf("failed to add child to node")
		}
		for _, ch := range testNode.Children {
			if equal := reflect.DeepEqual(ch, testChild); equal {
				break
			} else {
				t.Errorf("no child of %v the added child %v\n", testNode, testChild)
			}
		}
	})
}

func TestDeleteChild(t *testing.T){
	testNode := NewNode("parent", "testing")
	testChild := NewNode("child", "testing")
	t.Run("deleteChild", func(t *testing.T){
		testNode.addChild(testChild)
		testNode.addChild(testChild)
		testNode.addChild(testChild)

		err := testNode.deleteChild(0)
		if err != nil {
			t.Fatalf("deleting the child caused an error: %v", err)
		}
		if len(testNode.Children) != 2 {
			t.Errorf("failed to delete child from node")
		}
		err = testNode.deleteChild(1)
		if err != nil {
			t.Fatalf("deleting the child caused an error: %v", err)
		}
		if len(testNode.Children) == 2 {
			t.Errorf("failed to delete child at index 1 from node")
		}
		err = testNode.deleteChild(4)
		if err == nil {
			t.Errorf("expected out of bounds index here")
		}
	})
}

func TestRelation(t *testing.T){
	parentObject := tc.Object{
		Msg: tc.Msg{
			Handle: 10,
			Parent: tc.HandleRoot,
		},
	}
	childObject := tc.Object{
		Msg: tc.Msg{
			Handle: 11,
			Parent: 10,
		},
	}

	parent := NewNodeWithObject("parent","testing", parentObject)
	child := NewNodeWithObject("child","testing", childObject)
	childObject.Msg.Parent = 200
	child2 := NewNodeWithObject("child2","testing", childObject)
	t.Run("isChild", func(t *testing.T){
		if !parent.isChild(*child) {
			t.Errorf("child is not related to parent")
		}

		if !child.isChildOf(*parent) {
			t.Errorf("child is not related to parent")
		}
	})
	t.Run("isChildFail", func(t *testing.T){
		if parent.isChild(*child2) {
			t.Errorf("child should not be related to parent")
		}
		if child.isChild(*parent) {
			t.Errorf("child should not be related to parent")
		}
	})
}

func TestCompare(t *testing.T){
	node1Object := tc.Object{
		Msg: tc.Msg{
			Ifindex: 1,
			Handle: 11,
			Parent: 10,
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			HfscQOpt: &tc.HfscQOpt{
				DefCls: 1,
			},
		},
	}

	node2Object := tc.Object{
		Msg: tc.Msg{
			Ifindex: 1,
			Handle: 11,
			Parent: 10,
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			HfscQOpt: &tc.HfscQOpt{
				DefCls: 1,
			},
		},
	}

	target := uint32(10)
	node3Object := tc.Object{
		Msg: tc.Msg{
			Ifindex: 4,
			Handle: 11,
			Parent: 10,
		},
		Attribute: tc.Attribute{
			Kind: "fq_codel",
			FqCodel: &tc.FqCodel{
				Target: &target,
			},
		},
	}

	node1 := NewNodeWithObject("node1","testing", node1Object)
	node2 := NewNodeWithObject("node2","testing", node2Object)
	node3 := NewNodeWithObject("node3","testing", node3Object)

	t.Run("compareMsg", func(t *testing.T){
		if !node1.equalMsg(*node2) {
			t.Errorf("2 objects with the same Msg are not comparing correctly")
		}
		if node1.equalMsg(*node3) {
			t.Errorf("2 objects that should not be equal, have equal Msg")
		}
		node2.Object.Msg.Handle = 300
		node2.Object.Msg.Parent = 200
		node3.Object.Msg.Handle = 300
		node3.Object.Msg.Parent = 200
		if node1.equalMsg(*node2) {
			t.Errorf("2 objects that should not be equal, have equal Msg")
		}
		if node1.equalMsg(*node3) {
			t.Errorf("2 objects that should not be equal, have equal Msg")
		}
	})

	t.Run("compareKind", func(t *testing.T){
		if !node1.equalKind(*node2) {
			t.Errorf("2 objects with the same Attrs are not comparing as false")
		}

		if node1.equalKind(*node3) {
			t.Errorf("2 objects with the different Attrs are not comparing as true")
		}
	})
}
