package main

import (
	"fmt"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/spf13/viper"
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
	key := "filter.testing.specs"
	switch fl.Type {
	case "basic":
		basic := &tc.Basic{}
		viper.UnmarshalKey(key, &basic)
		viper.UnmarshalKey(key+".police", &basic.Police)
		viper.UnmarshalKey(key+".ematch", &basic.Ematch)

		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			basic.ClassID = &v
		}

		attrs = tc.Attribute{
			Kind:  fl.Type,
			Basic: basic,
		}
	case "matchall":
		matchall := &tc.Matchall{}
		viper.UnmarshalKey(key, &matchall)
		viper.UnmarshalKey(key+".actions", &matchall.Actions)
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			matchall.ClassID = &v
		}
		attrs = tc.Attribute{
			Kind:     fl.Type,
			Matchall: matchall,
		}
	// case "rsvp":
	//  	// not really supported at the moment
	//  	// TODO: discover how to properly parse the bytes parts of this config

	// 	rsvp := &tc.Rsvp{}
	// 	viper.UnmarshalKey(key, &rsvp)
	// 	viper.UnmarshalKey(key+".info", &rsvp.PInfo)
	// 	viper.UnmarshalKey(key+".police", &rsvp.Police)
	// 	if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
	// 		rsvp.ClassID = &v
	// 	}
	// 	attrs = tc.Attribute{
	// 		Kind: fl.Type,
	// 		Rsvp: rsvp,
	// 	}
	case "route4":
		route := &tc.Route4{}
		viper.UnmarshalKey(key, &route)
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			route.ClassID = &v
		}
		attrs = tc.Attribute{
			Kind:   fl.Type,
			Route4: route,
		}
	case "flow":
		flow := &tc.Flow{}
		viper.UnmarshalKey(key, &flow)
		attrs = tc.Attribute{
			Kind: fl.Type,
			Flow: flow,
		}
	case "flower":
		flower := &tc.Flower{}
		viper.UnmarshalKey(key, &flower)

		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			flower.ClassID = &v
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

		attrs = tc.Attribute{
			Kind:   fl.Type,
			Flower: flower,
		}
	case "fw":
		fw := &tc.Fw{}
		viper.UnmarshalKey(key, &fw)
		viper.UnmarshalKey(key+".police", &fw.Police)
		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			fw.ClassID = &v
		}

		attrs = tc.Attribute{
			Kind: fl.Type,
			Fw:   fw,
		}
	case "u32":
		u32 := &tc.U32{}
		viper.UnmarshalKey(key, &u32)

		viper.UnmarshalKey(key+".mark", &u32.Mark)
		viper.UnmarshalKey(key+".sel", &u32.Sel)
		viper.UnmarshalKey(key+".police", &u32.Police)
		viper.UnmarshalKey(key+".actions", &u32.Actions)

		if v, ok := handleMap[fl.Specs["classid"].(string)]; ok {
			u32.ClassID = &v
		}

		attrs = tc.Attribute{
			Kind: fl.Type,
			U32:  u32,
		}
	}
	return attrs, nil
}
