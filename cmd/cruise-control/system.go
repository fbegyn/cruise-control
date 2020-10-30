package main

import (
	"fmt"
	"os"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

func GetInterfaceNodes(tcnl *tc.Tc, interf uint32, handleMap map[string]uint32) (tr []*Node) {
	revHandleMap := make(map[uint32]string)
	for k, v := range handleMap {
		revHandleMap[v] = k
	}

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
	//filters, err := tcnl.Class().Get(&tc.Msg{
	//	Family:  unix.AF_UNSPEC,
	//	Ifindex: interf,
	//})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get filters")
	}

	for _, qd := range qdiscs {
		if qd.Ifindex != interf {
			continue
		}
		qdHandle := revHandleMap[qd.Handle]
		n := NewNodeWithObject(qdHandle, "qdisc", qd)
		tr = append(tr, n)
	}
	for _, cl := range classes {
		clHandle := revHandleMap[cl.Handle]
		n := NewNodeWithObject(clHandle, "class", cl)
		tr = append(tr, n)
	}
	//for _, fl := range filters {
	//	flHandle := fmt.Sprintf("%d", fl.Handle)
	//	n := NewNodeWithObject(flHandle, "filter", fl)
	//	tr = append(tr, n)
	//}
	return
}
