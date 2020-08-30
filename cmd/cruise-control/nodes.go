package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/florianl/go-tc"
)

type Node struct {
	Name     string
	Type     string
	Object   tc.Object
	Children []*Node
}

// NewNode creates a new node with the TC object embedded and sets the type of the node
// types: qdisc, class and filter
func NewNode(n, typ string) *Node {
	return &Node{
		Name:     n,
		Type:     typ,
		Children: []*Node{},
	}
}

// NewNodeWithObject creates a new node with the TC object embedded and sets the type of the node
// types: qdisc, class and filter
func NewNodeWithObject(n, typ string, object tc.Object) *Node {
	return &Node{
		Name:     n,
		Type:     typ,
		Object:   object,
		Children: []*Node{},
	}
}

// isChildOf checks if the current node is a child of node n
func (tr *Node) isChildOf(n *Node) bool {
	if tr.Object.Msg.Parent == n.Object.Handle {
		return true
	}
	return false
}

// isChild checks if the node n is a child of the current node
func (tr *Node) isChild(n *Node) bool {
	if n.Object.Msg.Parent == tr.Object.Handle {
		return true
	}
	return false
}

// addNode add a node to the current node
func (tr *Node) addNode(n *Node) {
	tr.Children = append(tr.Children, n)
}

// addToNode add current node to node n as child
func (tr *Node) addToNode(n *Node) {
	n.Children = append(n.Children, tr)
}

// addChild add node n to node if is child
func (tr *Node) addChild(n *Node) {
	if tr.isChild(n) {
		tr.addNode(n)
	}
}

// addChild add node to node n if child of node n
func (tr *Node) addIfChild(n *Node) {
	if tr.isChildOf(n) {
		tr.addToNode(n)
	}
}

// equalHeader checks if the metadata of 2 nodes are the same
func (tr *Node) equalHeader(n *Node) bool {
	if tr.Name == n.Name && tr.Type == n.Type {
		return true
	}
	return false
}

// equalObject checks if the object of the nodes are the same
func (tr *Node) equalObject(n *Node) bool {
	if reflect.DeepEqual(tr, n) {
		return true
	}
	return false
}

// equalNode checks if the header and object of the nodes are the same
// it ignores the children, these should be check sperately with the
// equalChildren function
func (tr *Node) equalNode(n *Node) bool {
	if tr.equalHeader(n) && tr.equalObject(n) {
		return true
	}
	return false
}

// equalChildren check if the children of the nodes are the same
func (tr *Node) equalChildren(n *Node) bool {
	for _, child := range tr.Children {
		for _, peer := range n.Children {
			if equalChild := child.equalNode(peer); !equalChild {
				return false
			}
		}
	}
	return true
}

// FindRootNode finds the TC object with a root handle from a set of TC objects
func FindRootNode(nodes []*Node) (n *Node, index int) {
	for i, v := range nodes {
		if v.Object.Msg.Parent == tc.HandleRoot {
			return v, i
		}
		continue
	}
	return nil, 0
}

// ComposeTree composes the tree based on an array of tree nodes
func ComposeTree(nodes []*Node) (tr *Node) {
	tr, index := FindRootNode(nodes)
	nodes = append(nodes[:index], nodes[index+1:]...)
	tr.ComposeChildren(nodes)
	return
}

// FindChildren looks for the children of a node in a set of TC objects. It returns a slice of the
// children, the leftover nodes (nodes that are not children) and a boolean to indicate if the node
// has children or not in the set.
func (tr *Node) FindChildren(nodes []*Node) (children []*Node, leftover []*Node, hasChild bool) {
	var left []*Node
	hasChild = false
	for _, v := range nodes {
		if tr.isChild(v) {
			hasChild = true
			children = append(children, v)
			continue
		}
		left = append(left, v)
	}
	return children, left, hasChild
}

// AddChildren iterates over a set of nodes and if the node is a child, adds that node to the
// children of the current node
func (tr *Node) AddChildren(nodes []*Node) {
	for _, v := range nodes {
		tr.addChild(v)
	}
}

// ComposeChildren will preform a recursive lookup for the children of a node in a set of nodes. When
// this is called on the root/start node, the goal is to construct the entire tree from this. It
// returns the nodes that are not part of the composed tree.
func (tr *Node) ComposeChildren(nodes []*Node) (leftover []*Node) {
	children, leftover, haschild := tr.FindChildren(nodes)
	nodes = leftover
	if haschild {
		tr.AddChildren(children)
	}
	for _, v := range tr.Children {
		leftover = v.ComposeChildren(nodes)
		nodes = leftover
	}
	return leftover
}

// ApplyNode applies the tc object contained in the node with the replace function. If the object
// does not exists, creates it
func (tr *Node) ApplyNode(tcnl *tc.Tc) {
	logger.Log("level", "INFO", "handle", tr.Object.Handle, "type", tr.Type, "msg", "applying TC object")
	switch tr.Type {
	case "qdisc":
		if err := tcnl.Qdisc().Replace(&tr.Object); err != nil {
			fmt.Fprintf(os.Stderr, "could not assign qdisc to %d: %v\n", tr.Object.Ifindex, err)
			return
		}
	case "class":
		if err := tcnl.Class().Replace(&tr.Object); err != nil {
			fmt.Fprintf(os.Stderr, "could not assign class to %d: %v\n", tr.Object.Ifindex, err)
			return
		}
	case "filter":
		if err := tcnl.Filter().Replace(&tr.Object); err != nil {
			fmt.Fprintf(os.Stderr, "could not assign filter to %d: %v\n", tr.Object.Ifindex, err)
			return
		}
	default:
		fmt.Fprintf(os.Stderr, "Unkown TC object type\n")
	}
	for _, v := range tr.Children {
		v.ApplyNode(tcnl)
	}
}

// DeleteNode deletes the parent node (and as a consequence all children nodes will also be deleted)
func (tr *Node) DeleteNode(tcnl *tc.Tc) {
	logger.Log("level", "INFO", "handle", tr.Object.Handle, "type", tr.Type, "msg", "deleting TC object")
	switch tr.Type {
	case "qdisc":
		if err := tcnl.Qdisc().Delete(&tr.Object); err != nil {
			fmt.Fprintf(os.Stderr, "could not delete qdisc from %d: %v\n", tr.Object.Ifindex, err)
			return
		}
	case "class":
		if err := tcnl.Class().Delete(&tr.Object); err != nil {
			fmt.Fprintf(os.Stderr, "could not delete class from %d: %v\n", tr.Object.Ifindex, err)
			return
		}
	}
	for _, v := range tr.Children {
		v.ApplyNode(tcnl)
	}
}
