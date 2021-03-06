package main

import (
	"fmt"
	"os"

	"github.com/florianl/go-tc"
)

// Node holds a node of the TC tree style structure.
type Node struct {
	Name     string
	Type     string
	Parent   string
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

// addNode add node n to the current node tr
func (tr *Node) addChild(n *Node) {
	tr.Children = append(tr.Children, n)
}

// addToNode add the current node to node n
func (tr *Node) addToParent(n *Node) {
	n.Children = append(n.Children, tr)
}

// deleteChild delete child at indx from node
func (tr *Node) deleteChild(index int) error {
	if len(tr.Children) - index < 0 {
		return fmt.Errorf("index %d out of bounds", index)
	}
	tr.Children = append(tr.Children[:index], tr.Children[index+1:]...)
	return nil
}

// isChild checks if the node n is a child of the current node
func (tr Node) isChild(n Node) bool {
	return n.Object.Msg.Parent == tr.Object.Handle
}

// isChildOf checks if the current node is a child of node n
func (tr Node) isChildOf(n Node) bool {
	return tr.Object.Msg.Parent == n.Object.Handle
}

// equalMsg checks if the metadata of 2 nodes are the same
func (tr Node) equalMsg(n Node) bool {
	equalHandle := (tr.Object.Msg.Handle == n.Object.Msg.Handle) 
	equalInterface := (tr.Object.Msg.Ifindex == n.Object.Msg.Ifindex) 
	equalParent := (tr.Object.Msg.Parent == n.Object.Msg.Parent)
	return equalHandle && equalInterface && equalParent
}

// equalKind checks if the object of the nodes are the same
// TODO: figure out a better compare between node objects
// returned trees from the kernel are modified with defaults
func (tr Node) equalKind(n Node) bool {
	return tr.Object.Attribute.Kind == n.Object.Attribute.Kind
}

func CompareSC(a, b tc.ServiceCurve) bool {
	return (a.D == b.D && a.M1 == b.M1 && a.M2 == b.M2)
}

func (tr Node) equalProperties(n Node) bool {
	switch tr.Object.Attribute.Kind {
	case "hfsc":	
		if tr.Object.Hfsc == nil && n.Object.Hfsc == nil { break }
		trHfsc := tr.Object.Hfsc
		nHfsc := n.Object.Hfsc
		m := make(map[string]bool)
		if trHfsc.Rsc != nil && nHfsc.Rsc != nil {
			m["rsc"] = CompareSC(*trHfsc.Rsc, *nHfsc.Rsc)
		}
		if trHfsc.Usc != nil && nHfsc.Usc != nil {
			m["usc"] = CompareSC(*trHfsc.Usc, *nHfsc.Usc)
		}
		if trHfsc.Fsc != nil && nHfsc.Fsc != nil {
			m["fsc"] = CompareSC(*trHfsc.Fsc, *nHfsc.Fsc)
		}

		for _, v := range m {
			if !v {
				return false
			}
		}
		return true
	}
	return true
}

// equalNode checks if the header and object of the nodes are the same
// it ignores the children, these should be check sperately with the
// equalChildren function
func (tr Node) equalNode(n Node) bool {
	tr.equalProperties(n)
	return (tr.equalMsg(n) && tr.equalKind(n) && tr.equalProperties(n))
}

// equalChildren check if the children of the nodes are the same
func (tr *Node) equalChildren(n *Node) bool {
	equalChildren := true
	for _, child := range tr.Children {
		equalChild := false
		for _, peer := range n.Children {
			if equalChild = child.equalNode(*peer); equalChild {
				break
			}
		}
		equalChildren = equalChildren && equalChild
		if !equalChildren {
			break
		}
	}
	return equalChildren
}

// AddChildren iterates over a set of nodes and if the node is a child, adds that node to the
// children of the current node
func (tr *Node) AddChildren(nodes []*Node) {
	for _, v := range nodes {
		if tr.isChild(*v) {
		    tr.addChild(v)
		}
	}
}

// FindChildren looks for the children of a node in a set of TC objects. It returns a slice of the
// children, the leftover nodes (nodes that are not children) and a boolean to indicate if the node
// has children or not in the set.
func (tr *Node) FindChildren(nodes []*Node) (children []*Node, leftover []*Node, hasChild bool) {
	var left []*Node
	hasChild = false
	for _, v := range nodes {
		if tr.isChild(*v) {
			hasChild = true
			children = append(children, v)
			continue
		}
		left = append(left, v)
	}
	return children, left, hasChild
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
	for _, v := range tr.Children {
		v.DeleteNode(tcnl)
	}
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
	case "filter":
		if err := tcnl.Filter().Delete(&tr.Object); err != nil {
			fmt.Fprintf(os.Stderr, "could not delete filter from %d: %v\n", tr.Object.Ifindex, err)
			return
		}
	}
}

// FindPeer finds the child of another node that matches the current selected node. Can be used to
// easily check if 2 nodes share the same child. Return the node and a boolean. If true, the returned
// node is the peer. If false, the returned node is the child itself
// TODO: implement the function
func (tr *Node) FindPeer(child, n *Node) (*Node, bool) {
	return nil, false
}

// FindRootNode finds the TC object with a root handle from a set of TC objects
func FindRootNode(nodes []*Node) (n *Node, index int) {
	for i, v := range nodes {
		if v.Object.Msg.Parent == tc.HandleRoot {
			return v, i
		}
	}
	return nil, 0
}

