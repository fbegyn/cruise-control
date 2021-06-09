package main

import (
	"fmt"
	"net"
	"os"

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
	actionMap map[string][]*tc.Action,
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
		attrs, err := parseFilterAttrs(filt, handleMap, actionMap)
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
func parseFilterAttrs(fl FilterConfig, handleMap map[string]uint32, actionMap map[string][]*tc.Action) (attrs tc.Attribute, err error) {
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
			Kind:  fl.Type,
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
			Kind:   fl.Type,
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
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			flower.ClassID = &v
		}
		if v, ok := fl.Specs["indev"].(string); ok {
			flower.Indev = &v
		}
		if v, ok := actionMap[fl.Specs["actions"].(string)]; ok {
			flower.Actions = &v
		}
		if v, ok := fl.Specs["keyethsrc"].(string); ok {
			temp, err := net.ParseMAC(v)
			if err != nil {
				os.Exit(10)
			}
			flower.KeyEthSrc = &temp
		}
		if v, ok := fl.Specs["keyethsrcmask"].(string); ok {
			temp, err := net.ParseMAC(v)
			if err != nil {
				os.Exit(10)
			}
			flower.KeyEthSrcMask = &temp
		}

		if v, ok := fl.Specs["keyethdst"].(string); ok {
			temp, err := net.ParseMAC(v)
			if err != nil {
				os.Exit(10)
			}
			flower.KeyEthDst = &temp
		}
		if v, ok := fl.Specs["keyethdstmask"].(string); ok {
			temp, err := net.ParseMAC(v)
			if err != nil {
				os.Exit(10)
			}
			flower.KeyEthDstMask = &temp
		}

		if v, ok := fl.Specs["keyethtype"].(int64); ok {
			temp := uint16(v)
			flower.KeyEthType = &temp
		}
		if v, ok := fl.Specs["keyipproto"].(int64); ok {
			temp := uint8(v)
			flower.KeyIPProto = &temp
		}

		ip4src, ipok := fl.Specs["keyipv4src"].(string)
		ip4srcmask, maskok := fl.Specs["keyipv4srcmask"].(int64)
		if ipok && maskok {
			cidr := fmt.Sprintf("%s/%d", ip4src, ip4srcmask)

			ip, net, err := net.ParseCIDR(cidr)
			if err != nil {
				os.Exit(10)
			}
			flower.KeyIPv4Src = &ip
			flower.KeyIPv4SrcMask = &net.Mask

		}

		ip4dst, ipok := fl.Specs["keyipv4dst"].(string)
		ip4dstmask, maskok := fl.Specs["keyipv4dstmask"].(int64)
		if ipok && maskok {
			cidr := fmt.Sprintf("%s/%d", ip4dst, ip4dstmask)

			ip, net, err := net.ParseCIDR(cidr)
			if err != nil {
				os.Exit(10)
			}
			flower.KeyIPv4Dst = &ip
			flower.KeyIPv4DstMask = &net.Mask

		}
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
		if v, ok := fl.Specs["keyvlanprio"].(int64); ok {
			temp := uint8(v)
			flower.KeyVlanPrio = &temp
		}
		if v, ok := fl.Specs["keyvlanethtype"].(int64); ok {
			temp := uint16(v)
			flower.KeyVlanEthType = &temp
		}
		if v, ok := fl.Specs["keyenckeyid"].(int64); ok {
			temp := uint32(v)
			flower.KeyEncKeyID = &temp
		}

		ip4encsrc, ipok := fl.Specs["keyipv4dst"].(string)
		ip4encsrcmask, maskok := fl.Specs["keyipv4dstmask"].(int64)
		if ipok && maskok {
			cidr := fmt.Sprintf("%s/%d", ip4encsrc, ip4encsrcmask)

			ip, net, err := net.ParseCIDR(cidr)
			if err != nil {
				os.Exit(10)
			}
			flower.KeyEncIPv4Src = &ip
			flower.KeyEncIPv4SrcMask = &net.Mask

		}

		ip4encdst, ipok := fl.Specs["keyencipv4dst"].(string)
		ip4encdstmask, maskok := fl.Specs["keyencipv4dstmask"].(int64)
		if ipok && maskok {
			cidr := fmt.Sprintf("%s/%d", ip4encdst, ip4encdstmask)

			ip, net, err := net.ParseCIDR(cidr)
			if err != nil {
				os.Exit(10)
			}
			flower.KeyEncIPv4Dst = &ip
			flower.KeyEncIPv4DstMask = &net.Mask

		}
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
		if v, ok := fl.Specs["keyencudpsrcport"].(int64); ok {
			temp := uint16(v)
			flower.KeyEncUDPSrcPort = &temp
		}
		if v, ok := fl.Specs["keyencudpsrcportmask"].(int64); ok {
			temp := uint16(v)
			flower.KeyEncUDPSrcPortMask = &temp
		}
		if v, ok := fl.Specs["keyencudpdstport"].(int64); ok {
			temp := uint16(v)
			flower.KeyEncUDPDstPort = &temp
		}
		if v, ok := fl.Specs["keyencudpdstportmask"].(int64); ok {
			temp := uint16(v)
			flower.KeyEncUDPDstPortMask = &temp
		}
		if v, ok := fl.Specs["keyflags"].(int64); ok {
			temp := uint32(v)
			flower.KeyFlags = &temp
		}
		if v, ok := fl.Specs["keyflagsmask"].(int64); ok {
			temp := uint32(v)
			flower.KeyFlagsMask = &temp
		}
		if v, ok := fl.Specs["keyicmpv4code"].(int64); ok {
			temp := uint8(v)
			flower.KeyIcmpv4Code = &temp
		}
		if v, ok := fl.Specs["keyicmpv4codemask"].(int64); ok {
			temp := uint8(v)
			flower.KeyIcmpv4CodeMask = &temp
		}
		if v, ok := fl.Specs["keyicmpv4type"].(int64); ok {
			temp := uint8(v)
			flower.KeyIcmpv4Type = &temp
		}
		if v, ok := fl.Specs["keyicmpv4typemask"].(int64); ok {
			temp := uint8(v)
			flower.KeyIcmpv4TypeMask = &temp
		}
		if v, ok := fl.Specs["keyicmpv6code"].(int64); ok {
			temp := uint8(v)
			flower.KeyIcmpv6Code = &temp
		}
		if v, ok := fl.Specs["keyicmpv6codemask"].(int64); ok {
			temp := uint8(v)
			flower.KeyIcmpv6CodeMask = &temp
		}
		if v, ok := fl.Specs["keyarpsip"].(int64); ok {
			temp := uint32(v)
			flower.KeyArpSIP = &temp
		}
		if v, ok := fl.Specs["keyarpsipmask"].(int64); ok {
			temp := uint32(v)
			flower.KeyArpSIPMask = &temp
		}
		if v, ok := fl.Specs["keyarptip"].(int64); ok {
			temp := uint32(v)
			flower.KeyArpTIP = &temp
		}
		if v, ok := fl.Specs["keyarptipmask"].(int64); ok {
			temp := uint32(v)
			flower.KeyArpTIPMask = &temp
		}
		if v, ok := fl.Specs["keyarpop"].(int64); ok {
			temp := uint8(v)
			flower.KeyArpOp = &temp
		}
		if v, ok := fl.Specs["keyarpopmask"].(int64); ok {
			temp := uint8(v)
			flower.KeyArpOpMask = &temp
		}
		if v, ok := fl.Specs["keymplsttl"].(int64); ok {
			temp := uint8(v)
			flower.KeyMplsTTL = &temp
		}
		if v, ok := fl.Specs["keymplsbos"].(int64); ok {
			temp := uint8(v)
			flower.KeyMplsBos = &temp
		}
		if v, ok := fl.Specs["keymplstc"].(int64); ok {
			temp := uint8(v)
			flower.KeyMplsTc = &temp
		}
		if v, ok := fl.Specs["keymplslabel"].(int64); ok {
			temp := uint32(v)
			flower.KeyMplsLabel = &temp
		}
		if v, ok := fl.Specs["keytcpflags"].(int64); ok {
			temp := uint16(v)
			flower.KeyTCPFlags = &temp
		}
		if v, ok := fl.Specs["keytcpflagsmask"].(int64); ok {
			temp := uint16(v)
			flower.KeyTCPFlagsMask = &temp
		}
		if v, ok := fl.Specs["keyiptos"].(int64); ok {
			temp := uint8(v)
			flower.KeyIPTOS = &temp
		}
		if v, ok := fl.Specs["keyiptosmask"].(int64); ok {
			temp := uint8(v)
			flower.KeyIPTOSMask = &temp
		}
		if v, ok := fl.Specs["keyipttl"].(int64); ok {
			temp := uint8(v)
			flower.KeyIPTTL = &temp
		}
		if v, ok := fl.Specs["keyipttlmask"].(int64); ok {
			temp := uint8(v)
			flower.KeyIPTTLMask = &temp
		}
		if v, ok := fl.Specs["flags"].(int64); ok {
			temp := uint32(v)
			flower.Flags = &temp
		}
		if v, ok := fl.Specs["keycvlanid"].(int64); ok {
			temp := uint16(v)
			flower.KeyCVlanID = &temp
		}
		if v, ok := fl.Specs["keycvlanprio"].(int64); ok {
			temp := uint8(v)
			flower.KeyCVlanPrio = &temp
		}
		if v, ok := fl.Specs["keycvlanethtype"].(int64); ok {
			temp := uint16(v)
			flower.KeyCVlanEthType = &temp
		}
		if v, ok := fl.Specs["keyenciptos"].(int64); ok {
			temp := uint8(v)
			flower.KeyEncIPTOS = &temp
		}
		if v, ok := fl.Specs["keyenciptosmask"].(int64); ok {
			temp := uint8(v)
			flower.KeyEncIPTOSMask = &temp
		}
		if v, ok := fl.Specs["keyencipttl"].(int64); ok {
			temp := uint8(v)
			flower.KeyEncIPTTL = &temp
		}
		if v, ok := fl.Specs["keyencipttlmask"].(int64); ok {
			temp := uint8(v)
			flower.KeyEncIPTTLMask = &temp
		}
		if v, ok := fl.Specs["inhwcount"].(int64); ok {
			temp := uint32(v)
			flower.InHwCount = &temp
		}
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
		if v, ok := fl.Specs["mask"].(int64); ok {
			temp := uint32(v)
			fw.Mask = &temp
		}
		attrs = tc.Attribute{
			Kind: fl.Type,
			Fw:   fw,
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
		if v, ok := fl.Specs["hash"].(int64); ok {
			temp := uint32(v)
			tcindex.Hash = &temp
		}
		if v, ok := fl.Specs["mask"].(int64); ok {
			temp := uint16(v)
			tcindex.Mask = &temp
		}
		if v, ok := fl.Specs["shift"].(int64); ok {
			temp := uint32(v)
			tcindex.Shift = &temp
		}
		if v, ok := fl.Specs["fallthrough"].(int64); ok {
			temp := uint32(v)
			tcindex.FallThrough = &temp
		}
	}
	return attrs, nil
}
