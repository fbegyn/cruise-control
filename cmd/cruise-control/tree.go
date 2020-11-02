package main

import (
	"github.com/florianl/go-tc"
)

// CompareTree validates if tr matches the tree of argument n
func (tr *Node) CompareTree(n *Node) bool {
	if !tr.equalNode(n) {
		return false
	}

	if len(tr.Children) != len(n.Children) {
		return false
	}

	equalChildren := true
	for _, child := range tr.Children {
		equalChild := false
		for _, peer := range n.Children {
			if equalChild = child.CompareTree(peer); equalChild {
				break
			}
		}
		equalChildren = equalChildren && equalChild
	}
	return equalChildren
}

// TODO: implement function
func (tr *Node) UpdateTree(n *Node, tcnl *tc.Tc) {
	if !tr.equalNode(n) {
		n.ApplyNode(tcnl)
		tr.DeleteNode(tcnl)
		return
	} else {
		n.ApplyNode(tcnl)
	}

	for _, child := range tr.Children {
		for _, peer := range n.Children {
			if child.CompareTree(peer) {
				peer.ApplyNode(tcnl)
				child.DeleteNode(tcnl)
				break
			}
		}
	}
}

// ComposeTree composes the tree based on an array of tree nodes
func ComposeTree(nodes []*Node) (tr *Node) {
	tr, index := FindRootNode(nodes)
	nodes = append(nodes[:index], nodes[index+1:]...)
	tr.ComposeChildren(nodes)
	return
}
