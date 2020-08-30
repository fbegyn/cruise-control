package main

import (
	"fmt"
	"os"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

func GetInterfaceNodes(tcnl *tc.Tc, interf uint32) (tr []*Node) {
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
		if qd.Ifindex != interf {
			continue
		}
		qdHandle := fmt.Sprintf("%d", qd.Handle)
		n := NewNodeWithObject(qdHandle, "qdisc", qd)
		tr = append(tr, n)
	}
	for _, cl := range classes {
		clHandle := fmt.Sprintf("%d", cl.Handle)
		n := NewNodeWithObject(clHandle, "class", cl)
		tr = append(tr, n)
	}
	for _, fl := range filters {
		flHandle := fmt.Sprintf("%d", fl.Handle)
		n := NewNodeWithObject(flHandle, "filter", fl)
		tr = append(tr, n)
	}
	return
}
