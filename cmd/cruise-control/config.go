package main

import (
	"fmt"

	"github.com/florianl/go-tc"
)

// Parse the qdiscs handles into their uint32 form. Returns a map that can be
// used to look up the handle of any qdisc.
func parseQdiscHandle(config map[string]QdiscConfig) (map[string]uint32, error) {
	handleMap := make(map[string]uint32)
	for k, v := range config {
		handleMap[k] = StrHandle(v.Handle)
	}
	return handleMap, nil
}

// Parse the classes handles (classID) into their uint32 form. Returns a map
// that can be used to look up the handle of any class.
func parseClassHandle(config map[string]ClassConfig) (map[string]uint32, error) {
	handleMap := make(map[string]uint32)
	for k, v := range config {
		handleMap[k] = StrHandle(v.ClassID)
	}
	return handleMap, nil
}

// Parse the filters handles into their uint32 form. Returns a map that can be
// used to look up the handle of any filter.
func parseFilterHandle(config map[string]FilterConfig) (map[string]uint32, error) {
	handleMap := make(map[string]uint32)
	for k, v := range config {
		handleMap[k] = StrHandle(v.FilterID)
	}
	return handleMap, nil
}

// Parse the handles into useable uint32 formates. Returns a map that can be used
// to look up the handle of any object
func parseHandles(conf Config) (map[string]uint32, error) {
	h1, err := parseQdiscHandle(conf.Qdiscs)
	if err != nil {
		return nil, err
	}
	h2, err := parseClassHandle(conf.Classes)
	if err != nil {
		return nil, err
	}
	h3, err := parseFilterHandle(conf.Filters)
	if err != nil {
		return nil, err
	}

	for k, v := range h2 {
		h1[k] = v
	}
	for k, v := range h3 {
		h1[k] = v
	}
	return h1, nil
}

// Parse the qdiscs parents into their uint32 form. Returns a map that can be
// used to look up the parent of any qdisc.
func parseQdiscParents(handleMap map[string]uint32, config map[string]QdiscConfig) (map[string]uint32, error) {
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
			return nil, fmt.Errorf("failed to lookup parent, are you sure of it?")
		}
	}
	return parentMap, nil
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
			return nil, fmt.Errorf("failed to lookup parent, are you sure of it?")
		}
	}
	return parentMap, nil
}

// Parse parents of a config into a map. This map can be used to lookup the
// parent of any object
func parseParents(handleMap map[string]uint32, conf Config) (map[string]uint32, error) {
	h1, err := parseQdiscParents(handleMap, conf.Qdiscs)
	if err != nil {
		return nil, err
	}
	h2, err := parseClassParents(handleMap, conf.Classes)
	if err != nil {
		return nil, err
	}

	for k, v := range h2 {
		h1[k] = v
	}
	return h1, nil
}

func composeQdiscs(qdiscsConfigs map[string]QdiscConfig) map[string]*tc.Attribute {
	qdMap := make(map[string]*tc.Attribute)
	for qdName, qd := range qdiscsConfigs {
		logger.Log("level", "INFO", "msg", "parsing qdisc", "name", qdName, "handle", qd.Handle, "type", qd.Type)
		qdiscAttrs, err := parseQdiscAttrs(qd)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
		} else {
			logger.Log("level", "INFO", "msg", "qdisc parsed and adding to map")
			qdMap[qdName] = qdiscAttrs
		}
	}
	return qdMap
}

func composeClasses(classConfigs map[string]ClassConfig) map[string]*tc.Attribute {
	clMap := make(map[string]*tc.Attribute)
	for clName, cl := range classConfigs {
		logger.Log("level", "INFO", "msg", "parsing clas", "name", clName, "handle", cl.ClassID, "type", cl.Type)
		classAttrs, err := parseClassAttrs(cl)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
		} else {
			logger.Log("level", "INFO", "msg", "class parsed and adding to map")
			clMap[clName] = classAttrs
		}
	}
	return clMap
}

func composeFilters(filterConfigs map[string]FilterConfig) map[string]*tc.Object {
	flMap := make(map[string]*tc.Object)
	for filtName, filt := range filterConfigs {
		logger.Log("level", "INFO", "msg", "parsing filter", "name", filtName, "filter", filt)
		//filter, err := parseFilter()
		//if err != nil {
		//	logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
		//} else {
		//	logger.Log("level", "INFO", "msg", "qdisc parsed and adding to map")
		//	flMap = append(flMap, filter)
		//}
	}
	return flMap
}

func parseQdiscAttrs(qd QdiscConfig) (attrs *tc.Attribute, err error) {
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
		attrs = &tc.Attribute{
			Kind:    qd.Type,
			FqCodel: fqcodel,
		}
	case "hfsc":
		hfsc := &tc.HfscQOpt{}
		if v, ok := qd.Specs["defcls"]; ok {
			hfsc.DefCls = uint16(v)
		}
		attrs = &tc.Attribute{
			Kind:     qd.Type,
			HfscQOpt: hfsc,
		}
	}
	return attrs, nil
}

func parseClassAttrs(cl ClassConfig) (attrs *tc.Attribute, err error) {
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
		attrs = &tc.Attribute{
			Kind: cl.Type,
			Hfsc: hfsc,
		}
	}
	return attrs, nil
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
