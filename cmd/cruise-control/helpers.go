package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

// StrHandle is a simple helper function that desctructs a human readable handle into a uint32 that
// can be passed to the go-tc
func StrHandle(handle string) (uint32, error) {
	if handle == "root" {
		return tc.HandleRoot, nil
	}
	handleParts := strings.Split(handle, ":")
	handleMaj, err := strconv.ParseInt(handleParts[0], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the major part of the handle: %s", err)
	}
	handleMin, err := strconv.ParseInt(handleParts[1], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the minor part of the handle: %s", err)
	}
	return core.BuildHandle(uint32(handleMaj), uint32(handleMin)), nil
}

// SetSC implements the SC from the `tc` CLI. This function behaves the same as if one would set the
// USC through the `tc` command-line tool. This means bandwidth (m1 and m2) is specified in bits and
// the delay in ms.
func SetSC(hfsc *tc.Hfsc, m1 uint32, d uint32, m2 uint32) {
	hfsc.Rsc.M1 = m1
	hfsc.Rsc.D = d
	hfsc.Rsc.M2 = m2
	hfsc.Fsc.M1 = m1
	hfsc.Fsc.D = d
	hfsc.Fsc.M2 = m2
}

// SetUL implements the UL from the `tc` CLI. This function behaves the same as if one would set the
// USC through the `tc` command-line tool. This means bandwidth (m1 and m2) is specified in bits and
// the delay in ms.
func SetUL(hfsc *tc.Hfsc, m1 uint32, d uint32, m2 uint32) {
	hfsc.Usc.M1 = m1
	hfsc.Usc.D = d
	hfsc.Usc.M2 = m2
}

// SetLS implements the LS from the `tc` CLI. This function behaves the same as if one would set the
// USC through the `tc` command-line tool. This means bandwidth (m1 and m2) is specified in bits and
// the delay in ms.
func SetLS(hfsc *tc.Hfsc, m1 uint32, d uint32, m2 uint32) {
	hfsc.Fsc.M1 = m1
	hfsc.Fsc.D = d
	hfsc.Fsc.M2 = m2
}

// SetRT implements the RT from the `tc` CLI. This function behaves the same as if one would set the
// RSC through the `tc` command-line tool. This means bandwidth (m1 and m2) is specified in bits and
// the delay in ms.
func SetRT(hfsc *tc.Hfsc, m1 uint32, d uint32, m2 uint32) {
	hfsc.Rsc.M1 = m1
	hfsc.Rsc.D = d
	hfsc.Rsc.M2 = m2
}

func CreateTcSocket() (*tc.Tc, error) {
	// open a go-tc socket
	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		return nil, err
	}
	err = rtnl.SetOption(netlink.ExtendedAcknowledge, true)
	if err != nil {
		return nil, err
	}
	return rtnl, nil
}

func TCToJson(obj tc.Object, tcType string) (JsonObject) {
	maj, min := core.SplitHandle(obj.Handle)
	jsonObj := JsonObject{
		Type:      tcType,
		Interface: fmt.Sprintf("%d", obj.Ifindex),
		Handle:    fmt.Sprintf("%d:%d", maj, min),
		Attr:      obj.Attribute,
	}
	if obj.Parent != 0 {
		maj, min = core.SplitHandle(obj.Parent)
		jsonObj.Parent = fmt.Sprintf("%d:%d", maj, min)
	}
	return jsonObj
}

func JsonToTC(payload JsonObject) (tc.Object, error) {
	interf, err := net.InterfaceByName(payload.Interface)
	if err != nil {
		return tc.Object{}, err
	}
	handle, err := StrHandle(payload.Handle)
	parent, err := StrHandle(payload.Parent)

	obj := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  handle,
			Parent:  parent,
		},
		Attribute: payload.Attr,
	}
	return obj, nil
}
