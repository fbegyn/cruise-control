package main

import (
	"fmt"
	"net"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

// Parse the classes handles (classID) into their uint32 form. Returns a map
// that can be used to look up the handle of any class.
func parseClassHandle(config map[string]ClassConfig) (map[string]uint32, error) {
	handleMap := make(map[string]uint32)
	for k, v := range config {
		handle, err := StrHandle(v.ClassID)
		handleMap[k] = handle
		if err != nil {
			return nil, err
		}
	}
	return handleMap, nil
}

// Parse the classes parents into their uint32 form. Returns a map that can be
// used to look up the parent of any class.
func parseClassParents(handleMap map[string]uint32, config map[string]ClassConfig) (map[string]uint32, error) {
	parentMap := make(map[string]uint32)
	for k, v := range config {
		if v.Parent == "root" {
			parentMap[k] = tc.HandleRoot
			continue
		}
		if val, ok := handleMap[v.Parent]; ok {
			parentMap[k] = val
			continue
		} else {
			return nil, fmt.Errorf("failed to lookup parent for %s, are you sure of it?", k)
		}
	}
	return parentMap, nil
}

// given a map of classes and their handles, contstruct a map with tc objects of those classes
func composeClasses(
	handleMap, parentMap map[string]uint32,
	classConfigs map[string]ClassConfig,
	interf *net.Interface,
) (map[string]*tc.Object, error) {
	clMap := make(map[string]*tc.Object)
	for clName, cl := range classConfigs {
		// lookup the handle and parent of the object. If we can't find it, we
		// skip this object and log it
		handle, parent, found := lookupObjectHandleParent(handleMap, parentMap, clName)
		if !found {
			logger.Log("level", "WARN", "msg", "failed to lookup handle and/or parent of %v", clName)
			continue
		}
		// tc.Msg object for the object. We can construct this purely of the
		// information we already have.
		msg := tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  handle,
			Parent:  parent,
		}

		// Find out the attributes of the class in question. We error out of here
		// when a class can't be parsed, since we assume there is an error in the
		// config at that point.
		logger.Log("level", "INFO", "msg", "parsing clas", "name", clName, "handle", cl.ClassID, "type", cl.Type)
		classAttrs, err := parseClassAttrs(cl)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
			return nil, fmt.Errorf("a qdisc failed to be parsed, maybe double check the config for %s", clName)
		}
		logger.Log("level", "INFO", "msg", "class parsed and adding to map")

		// construct the qdisc
		class := &tc.Object{
			Msg:       msg,
			Attribute: classAttrs,
		}
		clMap[clName] = class
	}
	return clMap, nil
}

// parse the class attributes from the config file
func parseClassAttrs(cl ClassConfig) (attrs tc.Attribute, err error) {
	switch cl.Type {
	case "hfsc":
		hfsc := &tc.Hfsc{
			Rsc: &tc.ServiceCurve{},
			Fsc: &tc.ServiceCurve{},
			Usc: &tc.ServiceCurve{},
		}
		for typ, params := range cl.Specs {
			burst := params.(map[string]interface{})["burst"].(float64)
			delay := params.(map[string]interface{})["delay"].(int64)
			rate := params.(map[string]interface{})["rate"].(float64)
			switch typ {
			case "sc":
				SetSC(hfsc, uint32(burst), uint32(delay), uint32(rate))
			case "ul":
				SetUL(hfsc, uint32(burst), uint32(delay), uint32(rate))
			case "ls":
				SetLS(hfsc, uint32(burst), uint32(delay), uint32(rate))
			case "rt":
				SetRT(hfsc, uint32(burst), uint32(delay), uint32(rate))
			}
		}
		attrs = tc.Attribute{
			Kind: cl.Type,
			Hfsc: hfsc,
		}
	}
	return attrs, nil
}

// HELPER functions below to make it easier to modify classes

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
