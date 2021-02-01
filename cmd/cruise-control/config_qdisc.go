package main

import (
	"fmt"
	"net"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

// Parse the qdiscs handles into their uint32 form. Returns a map that can be
// used to look up the handle of any qdisc.
func parseQdiscHandle(config map[string]QdiscConfig) (map[string]uint32, error) {
	// create a map for the string name to the uint32 representation
	handleMap := make(map[string]uint32)
	for k, v := range config {
		handle, err := StrHandle(v.Handle)
		handleMap[k] = handle
		if err != nil {
			return nil, err
		}
	}
	return handleMap, nil
}

// Parse the qdiscs parents into their uint32 form. Returns a map that can be
// used to look up the parent of any qdisc.
func parseQdiscParents(
	handleMap map[string]uint32,
	config map[string]QdiscConfig,
) (map[string]uint32, error) {
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

// constructs a map of tc.Objects of all the qdiscs defined in the configuration
func composeQdiscs(
	handleMap, parentMap map[string]uint32,
	qdiscsConfigs map[string]QdiscConfig,
	interf *net.Interface,
) (map[string]*tc.Object, error) {
	// construct the map with tc objects
	qdMap := make(map[string]*tc.Object)

	// iterate over all qdiscs defined in the config
	for qdName, qd := range qdiscsConfigs {
		// lookup the handle and parent of the object. If we can't find it, we
		// skip this object and log it
		handle, parent, found := lookupObjectHandleParent(handleMap, parentMap, qdName)
		if !found {
			logger.Log("level", "WARN", "msg", "failed to lookup handle and/or parent of %v", qdName)
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

		// Find out the attributes of the qdisc in question. We error out of here
		// when a qdisc can't be parsed, since we assume there is an error in the
		// config at that point.
		logger.Log("level", "INFO", "msg", "parsing qdisc", "name", qdName, "handle", qd.Handle, "type", qd.Type)
		qdiscAttrs, err := parseQdiscAttrs(qd)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
			return nil, fmt.Errorf("a qdisc failed to be parsed, maybe double check the config for %s", qdName)
		}
		logger.Log("level", "INFO", "msg", "qdisc parsed and adding to map")

		// construct the qdisc
		qdisc := &tc.Object{
			Msg:       msg,
			Attribute: qdiscAttrs,
		}
		qdMap[qdName] = qdisc
	}
	return qdMap, nil
}

// Parse the qdisc attributes from the config file
func parseQdiscAttrs(qd QdiscConfig) (attrs tc.Attribute, err error) {
	switch qd.Type {
	case "fq_codel":
		fqcodel := &tc.FqCodel{}
		if v, ok := qd.Specs["cethreshold"]; ok {
			fqcodel.CEThreshold = &v
		}
		if v, ok := qd.Specs["dropbatchsize"]; ok {
			fqcodel.DropBatchSize = &v
		}
		if v, ok := qd.Specs["ecn"]; ok {
			fqcodel.ECN = &v
		}
		if v, ok := qd.Specs["flows"]; ok {
			fqcodel.Flows = &v
		}
		if v, ok := qd.Specs["interval"]; ok {
			fqcodel.Interval = &v
		}
		if v, ok := qd.Specs["limit"]; ok {
			fqcodel.Limit = &v
		}
		if v, ok := qd.Specs["memorylimit"]; ok {
			fqcodel.MemoryLimit = &v
		}
		if v, ok := qd.Specs["quantum"]; ok {
			fqcodel.Quantum = &v
		}
		if v, ok := qd.Specs["target"]; ok {
			fqcodel.Target = &v
		}
		attrs = tc.Attribute{
			Kind:    qd.Type,
			FqCodel: fqcodel,
		}
	case "hfsc":
		hfsc := &tc.HfscQOpt{}
		if v, ok := qd.Specs["defcls"]; ok {
			hfsc.DefCls = uint16(v)
		}
		attrs = tc.Attribute{
			Kind:     qd.Type,
			HfscQOpt: hfsc,
		}
	}
	return attrs, nil
}
