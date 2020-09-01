package main

import (
	"net"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

func composeQdiscs(qdiscsConfigs map[string]QdiscConfig, interf *net.Interface) map[string]*tc.Object {
	qdMap := make(map[string]*tc.Object)
	for qdName, qd := range qdiscsConfigs {
		logger.Log("level", "INFO", "msg", "parsing qdisc", "name", qdName, "handle", qd.Handle, "type", qd.Type)
		qdisc, err := parseQdisc(StrHandle(qd.Handle), StrHandle(qd.Parent), uint32(interf.Index), qd)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
		} else {
			logger.Log("level", "INFO", "msg", "qdisc parsed and adding to map")
			qdMap[qdName] = qdisc
		}
	}
	return qdMap
}

func composeClasses(classConfigs map[string]ClassConfig, interf *net.Interface, downSpeed float64) map[string]*tc.Object {
	clMap := make(map[string]*tc.Object)
	for clName, cl := range classConfigs {
		logger.Log("level", "INFO", "msg", "parsing clas", "name", clName, "handle", cl.ClassID, "type", cl.Type)
		class, err := parseClass(StrHandle(cl.ClassID), StrHandle(cl.Parent), uint32(interf.Index), cl)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
		} else {
			logger.Log("level", "INFO", "msg", "qdisc parsed and adding to map")
			clMap[clName] = class
		}
	}
	return clMap
}

func composeFilters(classConfigs map[string]ClassConfig, interf *net.Interface, downSpeed float64) map[string]*tc.Object {
	flMap := make(map[string]*tc.Object)
	for filtName, filt := range classConfigs {
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

func parseQdisc(handle, parent uint32, index uint32, qd QdiscConfig) (*tc.Object, error) {
	var attrs tc.Attribute
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
	qdisc := &tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: index,
			Handle:  handle,
			Parent:  parent,
			Info:    2,
		},
		Attribute: attrs,
	}
	return qdisc, nil
}

func parseClass(handle, parent uint32, index uint32, cl ClassConfig) (*tc.Object, error) {
	var attrs tc.Attribute
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
	class := &tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: index,
			Handle:  handle,
			Parent:  parent,
			Info:    0,
		},
		Attribute: attrs,
	}
	return class, nil
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
