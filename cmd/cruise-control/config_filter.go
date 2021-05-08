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
		fmt.Println(fl.Specs)
		if v, ok := fl.Specs["classid"].(string); ok {
			temp := handleMap[v]
			flower.ClassID = &temp
		}
		if v, ok := fl.Specs["indev"].(string); ok {
			flower.Indev = &v
		}
		// Actions = 2            
		if v, ok := fl.Specs["keyethtype"].(int64); ok {
			temp := uint16(v)
			flower.KeyEthType = &temp
		}
		if v, ok := fl.Specs["keyipproto"].(uint8); ok {
			flower.KeyIPProto = &v
		}
		if v, ok := fl.Specs["keyipv4src"].(string); ok {
			temp := net.ParseIP(v)
			flower.KeyIPv4Src = &temp
		}
		//if v, ok := fl.Specs["KeyIPv4SrcMask"].(uint32); ok {

		//}
		if v, ok := fl.Specs["keyipv4dst"].(string); ok {
			temp := net.ParseIP(v)
			flower.KeyIPv4Dst = &temp
		}
		//if v, ok := fl.Specs["KeyIPv4DstMask"].(uint32); ok {
		//}
		if v, ok := fl.Specs["keytcpsrc"].(int64); ok {
			temp := uint16(v)
			flower.KeyTCPSrc = &temp
		}
		if v, ok := fl.Specs["keytcpdst"].(int64); ok {
			temp := uint16(v)
			flower.KeyTCPDst = &temp
		}
		if v, ok := fl.Specs["keyudpsrc"].(int64); ok {
			temp := uint16(v)
			flower.KeyUDPSrc = &temp
		}
		if v, ok := fl.Specs["keyudpdst"].(int64); ok {
			temp := uint16(v)
			flower.KeyUDPDst = &temp
		}
		if v, ok := fl.Specs["keyvlanid"].(int64); ok {
			temp := uint16(v)
			flower.KeyVlanID = &temp
		}
		if v, ok := fl.Specs["keyvlanprio"].(uint8); ok {
			temp := uint8(v)
			flower.KeyVlanPrio = &temp
		}
		if v, ok := fl.Specs["keyvlanethtype"].(int64); ok {
			temp := uint16(v)
			flower.KeyVlanEthType = &temp
		}
		if v, ok := fl.Specs["keyenckeyid"].(uint32); ok {
			temp := uint32(v)
			flower.KeyEncKeyID = &temp
		}
		if v, ok := fl.Specs["keyencipv4src"].(string); ok {
			ip := net.ParseIP(v)
			flower.KeyEncIPv4Src = &ip
		}
		//if v, ok := fl.Specs["KeyEncIPv4SrcMask"]; ok {
		//	
		//}
		if v, ok := fl.Specs["keyencipv4dst"].(string); ok {
			ip := net.ParseIP(v)
			flower.KeyEncIPv4Dst = &ip
		}
		//if v, ok := fl.Specs["KeyEncIPv4DstMask"]; ok {
		//	
		//}
		if v, ok := fl.Specs["keytcpsrcmask"].(int64); ok {
			temp := uint16(v)
			flower.KeyTCPSrcMask = &temp
		}
		if v, ok := fl.Specs["keytcpdstmask"].(int64); ok {
			temp := uint16(v)
			flower.KeyTCPDstMask = &temp
		}
		if v, ok := fl.Specs["keyudpsrcmask"].(int64); ok {
			temp := uint16(v)
			flower.KeyUDPSrcMask = &temp
		}
		if v, ok := fl.Specs["keyudpdstmask"].(int64); ok {
			temp := uint16(v)
			flower.KeyUDPDstMask = &temp
		}
		if v, ok := fl.Specs["keysctpsrc"].(int64); ok {
			temp := uint16(v)
			flower.KeySctpSrc = &temp
		}
		if v, ok := fl.Specs["keysctpdst"].(int64); ok {
			temp := uint16(v)
			flower.KeySctpDst = &temp
		}
		//if v, ok := fl.Specs["KeyEncUDPSrcPort"].(uint16); ok {
		//	flower.KeyEncUDPSrcPort = &temp
		//}
		//if v, ok := fl.Specs["KeyEncUDPSrcPortMask"].(uint16); ok {
		//	flower.KeyEncUDPSrcPortMask = &temp
		//}
		//if v, ok := fl.Specs["KeyEncUDPDstPort"].(uint16); ok {
		//	flower.KeyEncUDPDstPort = &temp
		//}
		//if v, ok := fl.Specs["KeyEncUDPDstPortMask"].(uint16); ok {
		//	flower.KeyEncUDPDstPortMask = &temp
		//}
		//if v, ok := fl.Specs["KeyFlags"].(uint32); ok {
		//	flower.KeyFlags = &temp
		//}
		//if v, ok := fl.Specs["KeyFlagsMask"].(uint32); ok {
		//	flower.KeyFlagsMask = &temp
		//}
		//if v, ok := fl.Specs["KeyIcmpv4Code"].(uint8); ok {
		//	flower.KeyIcmpv4Code = &temp
		//}
		//if v, ok := fl.Specs["KeyIcmpv4CodeMask"].(uint8); ok {
		//	flower.KeyIcmpv4CodeMask = &temp
		//}
		//if v, ok := fl.Specs["KeyIcmpv4Type"].(uint8); ok {
		//	flower.KeyIcmpv4Type = &temp
		//}
		//if v, ok := fl.Specs["KeyIcmpv4TypeMask"].(uint8); ok {
		//	flower.KeyIcmpv4TypeMask = &temp
		//}
		//if v, ok := fl.Specs["KeyIcmpv6Code"].(uint8); ok {
		//	flower.KeyIcmpv6Code = &temp
		//}
		//if v, ok := fl.Specs["KeyIcmpv6CodeMask"].(uint8); ok {
		//	flower.KeyIcmpv6CodeMask = &temp
		//}
		//if v, ok := fl.Specs["KeyArpSIP"].(uint32); ok {
		//	flower.KeyArpSIP = &temp
		//}
		//if v, ok := fl.Specs["KeyArpSIPMask"].(uint32); ok {
		//	flower.KeyArpSIPMask = &temp
		//}
		//if v, ok := fl.Specs["KeyArpTIP"].(uint32); ok {
		//	flower.KeyArpTIP = &temp
		//}
		//if v, ok := fl.Specs["KeyArpTIPMask"].(uint32); ok {
		//	flower.KeyArpTIPMask = &temp
		//}
		//if v, ok := fl.Specs["KeyArpOp"].(uint8); ok {
		//	flower.KeyArpOp = &temp
		//}
		//if v, ok := fl.Specs["KeyArpOpMask"].(uint8); ok {
		//	flower.KeyArpOpMask = &temp
		//}
		//if v, ok := fl.Specs["KeyMplsTTL"].(uint8); ok {
		//	flower.KeyMplsTTL = &temp
		//}
		//if v, ok := fl.Specs["KeyMplsBos"].(uint8); ok {
		//	flower.KeyMplsBos = &temp
		//}
		//if v, ok := fl.Specs["KeyMplsTc"].(uint8); ok {
		//	flower.KeyMplsTc = &temp
		//}
		//if v, ok := fl.Specs["KeyMplsLabel"].(uint32); ok {
		//	flower.KeyMplsLabel = &temp
		//}
		//if v, ok := fl.Specs["KeyTCPFlags"].(uint16); ok {
		//	flower.KeyTCPFlags = &temp
		//}
		//if v, ok := fl.Specs["KeyTCPFlagsMask"].(uint16); ok {
		//	flower.KeyTCPFlagsMask = &temp
		//}
		//if v, ok := fl.Specs["KeyIPTOS"].(uint8); ok {
		//	flower.KeyIPTOS = &temp
		//}
		//if v, ok := fl.Specs["KeyIPTOSMask"].(uint8); ok {
		//	flower.KeyIPTOSMask = &temp
		//}
		//if v, ok := fl.Specs["KeyIPTTL"].(uint8); ok {
		//	flower.KeyIPTTL = &temp
		//}
		//if v, ok := fl.Specs["KeyIPTTLMask"].(uint8); ok {
		//	flower.KeyIPTTLMask = &temp
		//}
		//if v, ok := fl.Specs["KeyCVlanID"].(uint16); ok {
		//	flower.KeyCVlanID = &temp
		//}
		//if v, ok := fl.Specs["KeyCVlanPrio"].(uint8); ok {
		//	flower.KeyCVlanPrio = &temp
		//}
		//if v, ok := fl.Specs["KeyCVlanEthType"].(uint16); ok {
		//	flower.KeyCVlanEthType = &temp
		//}
		//if v, ok := fl.Specs["KeyEncIPTOS"].(uint8); ok {
		//	flower.KeyEncIPTOS = &temp
		//}
		//if v, ok := fl.Specs["KeyEncIPTOSMask"].(uint8); ok {
		//	flower.KeyEncIPTOSMask = &temp
		//}
		//if v, ok := fl.Specs["KeyEncIPTTL"].(uint8); ok {
		//	flower.KeyEncIPTTL = &temp
		//}
		//if v, ok := fl.Specs["keyencipttlmask"].(uint8); ok {
		//	flower.KeyEncIPTTLMask = &temp
		//}
		//if v, ok := fl.Specs["inhwcount"].(uint32); ok {
		//	flower.InHwCount = &temp
		//}
		fmt.Println(flower)
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
