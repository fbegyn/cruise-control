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
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  handle,
			Parent:  parent,
		}
		attrs, err := parseFilterAttrs(filt, handleMap)
		filter := &tc.Object{
			Msg:       msg,
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
	switch fl.Type {
	case "basic":
		basic := &tc.Basic{}
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			basic.ClassID = &v
		}
		if v, ok := fl.Specs["match"].(tc.Police); ok {
			basic.Police = &v
		}
		attrs = tc.Attribute{
			Kind: fl.Type,
			Basic: basic,
		}
	case "route":
		route := &tc.Route4{}
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			route.ClassID = &v
		}
		if v, ok := fl.Specs["to"].(int64); ok {
			temp := uint32(v)
			route.To = &temp
		}
		if v, ok := fl.Specs["from"].(int64); ok {
			temp := uint32(v)
			route.From = &temp
		}

		if v, ok := fl.Specs["iif"].(int64); ok {
			temp := uint32(v)
			route.IIf = &temp
		}
		attrs = tc.Attribute{
			Kind: fl.Type,
			Route4: route,
		}
	case "flow":
		flow := &tc.Flow{}
		if v, ok := fl.Specs["keys"].(int64); ok {
			temp := uint32(v)
			flow.Keys = &temp
		}
		if v, ok := fl.Specs["mode"].(int64); ok {
			temp := uint32(v)
			flow.Mode = &temp
		}
		if v, ok := fl.Specs["baseclass"].(int64); ok {
			temp := uint32(v)
			flow.BaseClass = &temp
		}
		if v, ok := fl.Specs["rshift"].(int64); ok {
			temp := uint32(v)
			flow.RShift = &temp
		}
		if v, ok := fl.Specs["addend"].(int64); ok {
			temp := uint32(v)
			flow.Addend = &temp
		}
		if v, ok := fl.Specs["mask"].(int64); ok {
			temp := uint32(v)
			flow.Mask = &temp
		}
		if v, ok := fl.Specs["xor"].(int64); ok {
			temp := uint32(v)
			flow.XOR = &temp
		}
		if v, ok := fl.Specs["divisor"].(int64); ok {
			temp := uint32(v)
			flow.Divisor = &temp
		}
		if v, ok := fl.Specs["perturb"].(int64); ok {
			temp := uint32(v)
			flow.PerTurb = &temp
		}
		attrs = tc.Attribute{
			Kind: fl.Type,
			Flow: flow,
		}
	case "flower":
		flower := &tc.Flower{}
		if v, ok := fl.Specs["ClassID"]; ok {
			
		}
		if v, ok := fl.Specs["Indev"].(string); ok {
			flower.Indev = &v
		}
		// Actions = 2            
		if v, ok := fl.Specs["KeyEthType"].(uint16); ok {
			flower.KeyEthType = &v
		}
		if v, ok := fl.Specs["KeyIPProto"].(uint8); ok {
			flower.KeyIPProto = &v
		}
		if v, ok := fl.Specs["KeyIPv4Src"].(string); ok {
			temp := net.ParseIP(v)
			flower.KeyIPv4Src = &temp
		}
		if v, ok := fl.Specs["KeyIPv4SrcMask"].(uint32); ok {

		}
		if v, ok := fl.Specs["KeyIPv4Dst"].(string); ok {
			temp := net.ParseIP(v)
			flower.KeyIPv4Dst = &temp
		}
		if v, ok := fl.Specs["KeyIPv4DstMask"].(uint32); ok {
			
		}
		if v, ok := fl.Specs["KeyTCPSrc"].(uint16); ok {
			flower.KeyTCPSrc = &v
		}
		if v, ok := fl.Specs["KeyTCPDst"].(uint16); ok {
			flower.KeyTCPDst = &v
		}
		if v, ok := fl.Specs["KeyUDPSrc"].(uint16); ok {
			flower.KeyUDPSrc = &v
		}
		if v, ok := fl.Specs["KeyUDPDst"].(uint16); ok {
			flower.KeyUDPDst = &v
		}
		if v, ok := fl.Specs["KeyVlanID"].(uint16); ok {
			flower.KeyVlanID = &v
		}
		if v, ok := fl.Specs["KeyVlanPrio"].(uint8); ok {
			flower.KeyVlanPrio = &v
		}
		if v, ok := fl.Specs["KeyVlanEthType"].(uint16); ok {
			flower.KeyVlanEthType = &v
		}
		if v, ok := fl.Specs["KeyEncKeyID"].(uint32); ok {
			flower.KeyEncKeyID = &v
		}
		if v, ok := fl.Specs["KeyEncIPv4Src"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncIPv4SrcMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncIPv4Dst"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncIPv4DstMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyTCPSrcMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyTCPDstMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyUDPSrcMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyUDPDstMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeySctpSrc"]; ok {
			
		}
		if v, ok := fl.Specs["KeySctpDst"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncUDPSrcPort"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncUDPSrcPortMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncUDPDstPort"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncUDPDstPortMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyFlags"]; ok {
			
		}
		if v, ok := fl.Specs["KeyFlagsMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIcmpv4Code"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIcmpv4CodeMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIcmpv4Type"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIcmpv4TypeMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIcmpv6Code"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIcmpv6CodeMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyArpSIP"]; ok {
			
		}
		if v, ok := fl.Specs["KeyArpSIPMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyArpTIP"]; ok {
			
		}
		if v, ok := fl.Specs["KeyArpTIPMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyArpOp"]; ok {
			
		}
		if v, ok := fl.Specs["KeyArpOpMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyMplsTTL"]; ok {
			
		}
		if v, ok := fl.Specs["KeyMplsBos"]; ok {
			
		}
		if v, ok := fl.Specs["KeyMplsTc"]; ok {
			
		}
		if v, ok := fl.Specs["KeyMplsLabel"]; ok {
			
		}
		if v, ok := fl.Specs["KeyTCPFlags"]; ok {
			
		}
		if v, ok := fl.Specs["KeyTCPFlagsMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIPTOS"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIPTOSMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIPTTL"]; ok {
			
		}
		if v, ok := fl.Specs["KeyIPTTLMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyCVlanID"]; ok {
			
		}
		if v, ok := fl.Specs["KeyCVlanPrio"]; ok {
			
		}
		if v, ok := fl.Specs["KeyCVlanEthType"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncIPTOS"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncIPTOSMask"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncIPTTL"]; ok {
			
		}
		if v, ok := fl.Specs["KeyEncIPTTLMask"]; ok {
			
		}
	case "fw":
		fw := &tc.Fw{}
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			fw.ClassID = &v
		}
		if v, ok := fl.Specs["police"].(tc.Police); ok {
			fw.Police = &v
		}
		if v, ok := fl.Specs["inDev"].(string); ok {
			fw.InDev = &v
		}
		if v, ok := fl.Specs["mask"].(uint32); ok {
			fw.Mask = &v
		}
		attrs = tc.Attribute{
			Kind: fl.Type,
			Fw: fw,
		}
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
			U32:  u32,
		}
	case "tcindex":
		tcindex := &tc.TcIndex{}
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			tcindex.ClassID = &v
		}
		if v, ok := fl.Specs["hash"].(uint32); ok {
			tcindex.Hash = &v
		}
		if v, ok := fl.Specs["mask"].(uint16); ok {
			tcindex.Mask = &v
		}
		if v, ok := fl.Specs["shift"].(uint32); ok {
			tcindex.Shift = &v
		}
		if v, ok := fl.Specs["fallthrough"].(uint32); ok {
			tcindex.FallThrough = &v
		}
	}
	return attrs, nil
}
