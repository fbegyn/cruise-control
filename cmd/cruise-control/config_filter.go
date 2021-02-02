package main

import (
	"fmt"
	"net"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

// Parse the filters handles into their uint32 form. Returns a map that can be
// used to look up the handle of any filter.
func parseFilterHandle(config map[string]FilterConfig) (map[string]uint32, error) {
	handleMap := make(map[string]uint32)
	for k, v := range config {
		handle, err := StrHandle(v.FilterID)
		handleMap[k] = handle
		if err != nil {
			return nil, err
		}
	}
	return handleMap, nil
}

// Parse the qdiscs parents into their uint32 form. Returns a map that can be
// used to look up the parent of any qdisc.
func parseFilterParents(
	handleMap map[string]uint32,
	config map[string]FilterConfig,
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


// given a map of filters, contstruct a map with tc objects of those filters
func composeFilters(
	handleMap, parentMap map[string]uint32,
	filterConfigs map[string]FilterConfig,
	interf *net.Interface,
) (map[string]*tc.Object, error) {
	flMap := make(map[string]*tc.Object)
	for filtName, filt := range filterConfigs {
		handle, parent, found := lookupObjectHandleParent(handleMap, parentMap, filtName)
		if !found {
			logger.Log("level", "ERROR", "msg", "filter not found in handles")
			return nil, fmt.Errorf("filter handle lookup failed")
		}
		logger.Log("level", "INFO", "msg", "parsing filter", "name", filtName, "handle", handle, "parent", 0)
		msg := tc.Msg{
			Family: unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle: handle,
			Parent: parent,
		}
		attrs, err := parseFilterAttrs(filt, handleMap)
		filter := &tc.Object{
			Msg: msg,
			Attribute: attrs,
		}
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse filter")
			return nil, fmt.Errorf("filter failed to be parsed")
		} else {
			logger.Log("level", "INFO", "msg", "filter parsed and adding to map")
			flMap[filtName] = filter
		}
	}
	return flMap, nil
}

// parse the filter attributes from the config file
func parseFilterAttrs(fl FilterConfig, handleMap map[string]uint32) (attrs tc.Attribute, err error) {
	fmt.Println(fl)
	switch fl.Type {
	case "u32":
		u32 := &tc.U32{}
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			u32.ClassID = &v
		}
		switch fl.Specs["match"].(string) {
		case "mark":
			mark := &tc.U32Mark{}
			if v, ok := fl.Specs["mask"].(int64); ok {
				mark.Mask = uint32(v)
			}
			if v, ok := fl.Specs["val"].(int64); ok {
				mark.Val = uint32(v)
			}
			u32.Mark = mark
		}
		attrs = tc.Attribute{
			Kind: fl.Type,
			U32: u32,
		}
		fmt.Println(attrs.U32.ClassID)
		fmt.Println(attrs.U32.Mark)
	}
	return attrs, nil
}
