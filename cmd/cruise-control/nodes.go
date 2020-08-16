package main

import (
	"fmt"
	"os"

	"github.com/florianl/go-tc"
)

type Node struct {
	Type     string
	Handle   uint32
	Object   *tc.Object
	Children []*Node
}

func FindRootNode(nodes []*Node) (n *Node, index int) {
	for i, v := range nodes {
		if v.Object.Msg.Parent == tc.HandleRoot {
			return v, i
		}
		continue
	}
	return nil, 0
}

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

func (tr *Node) AddChildren(nodes []*Node) {
	for _, v := range nodes {
		tr.addChild(v)
	}
}

func (tr *Node) ComposeChildren(nodes []*Node) (leftover []*Node) {
	children, leftover, haschild := tr.FindChildren(nodes)
	nodes = leftover
	if haschild {
		tr.AddChildren(children)
	}
	for _, v := range tr.Children {
		leftover := v.ComposeChildren(nodes)
		nodes = leftover
	}
	return leftover
}

// ApplyNode applies the tc object contained in the node with the replace function. If the object
// does not exists, it creates it
func (tr *Node) ApplyNode(tcnl *tc.Tc) {
	logger.Log("level", "INFO", "handle", tr.Handle, "type", tr.Type, "msg", "applying TC object")
	switch tr.Type {
	case "qdisc":
		if err := tcnl.Qdisc().Replace(tr.Object); err != nil {
			fmt.Fprintf(os.Stderr, "could not assign qdisc to %d: %v\n", tr.Object.Ifindex, err)
			return
		}
	case "class":
		if err := tcnl.Class().Replace(tr.Object); err != nil {
			fmt.Fprintf(os.Stderr, "could not assign class to %d: %v\n", tr.Object.Ifindex, err)
			return
		}
	}
	for _, v := range tr.Children {
		v.ApplyNode(tcnl)
	}
}

func NewNode(object *tc.Object, typ string) *Node {
	return &Node{
		Type:     typ,
		Handle:   object.Msg.Handle,
		Object:   object,
		Children: []*Node{},
	}
}

// isChildOf checks if the current node is a child of node n
func (tr *Node) isChildOf(n *Node) bool {
	if tr.Object.Msg.Parent == n.Handle {
		return true
	}
	return false
}

// isChild checks if the node n is a child of the current node
func (tr *Node) isChild(n *Node) bool {
	if n.Object.Msg.Parent == tr.Handle {
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
