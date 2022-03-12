package main

import (
	"fmt"
	"os"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

func GetInterfaceNodes(tcnl *tc.Tc, interf uint32) (tr, filterNodes []*Node) {
	qdiscs, err := tcnl.Qdisc().Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get qdiscs")
	}
	classes, err := tcnl.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: interf,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get classes")
	}
	filters, err := tcnl.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: interf,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get filters")
	}

	for _, qd := range qdiscs {
		n := NewNodeWithObject("qdisc", qd)
		tr = append(tr, n)
	}
	for _, cl := range classes {
		n := NewNodeWithObject("class", cl)
		tr = append(tr, n)
	}

	for _, fl := range filters {
		n := NewNodeWithObject("filter", fl)
		filterNodes = append(filterNodes, n)
	}
	return
}
