package main

import (
	"fmt"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"
)

type Config struct {
	DownloadSpeed uint32
	UploadSpeed   uint32

	Interface string

	Qdiscs  map[string]Qdisc
	Classes map[string]Class
	Filters map[string]Filter
}

type Qdisc struct {
	Type   string
	Parent string
	Specs  map[string]uint32
}

type Class struct {
	Type   string
	Parent string
	Specs  map[string]interface{}
}

type Filter struct {
	Type  string
	Specs map[string]uint32
}

var logger log.Logger

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

	if err := viper.ReadInConfig(); err != nil {
		logger.Log("level", "ERROR", "msg", "failed to read config file", "error", err)
	}

	t := Config{}
	viper.Unmarshal(&t)
	fmt.Println(t)

	// determine the interface for cruise control to run on
	interf, err := net.InterfaceByName(t.Interface)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to parse interface")
	}

	for handle, qd := range t.Qdiscs {
		logger.Log("handle", handle, "type", qd.Type)
		parseQdisc(StrHandle(handle), StrHandle(qd.Parent), uint32(interf.Index), qd)
		//fmt.Println(test)
		//if err := rtnl.Qdisc().Add(&test); err != nil {
		//	fmt.Fprintf(os.Stderr, "could not assign %s to %s: %v %v\n", handle, t.Interface, err, test.Attribute.FqCodel)
		//	return
		//}
	}

	for handle, cl := range t.Classes {
		logger.Log("handle", handle, "type", cl.Type)
		parseClass(StrHandle(handle), StrHandle(cl.Parent), uint32(interf.Index), cl)
		//if err := rtnl.Qdisc().Add(&test); err != nil {
		//	fmt.Fprintf(os.Stderr, "could not assign %s to %s: %v %v\n", handle, t.Interface, err, test.Attribute.FqCodel)
		//	return
		//}
	}
}

func parseQdisc(handle, parent uint32, index uint32, qd Qdisc) (tc.Object, error) {
	var attrs tc.Attribute
	switch qd.Type {
	case "fq_codel":
		logger.Log("msg", "creating fq_codel qdisc")
		fqcodel := &tc.FqCodel{}
		fqcodel.CEThreshold = qd.Specs["cethreshold"]
		fqcodel.DropBatchSize = qd.Specs["dropbatchsize"]
		fqcodel.ECN = qd.Specs["ecn"]
		fqcodel.Flows = qd.Specs["flows"]
		fqcodel.Interval = qd.Specs["interval"]
		fqcodel.Limit = qd.Specs["limit"]
		fqcodel.MemoryLimit = qd.Specs["memorylimit"]
		fqcodel.Quantum = qd.Specs["quantum"]
		fqcodel.Target = qd.Specs["target"]
		attrs = tc.Attribute{
			Kind:    qd.Type,
			FqCodel: fqcodel,
		}
	case "hfsc":
		hfsc := &tc.HfscQOpt{}
		hfsc.DefCls = uint16(qd.Specs["defcls"])
		attrs = tc.Attribute{
			Kind:     qd.Type,
			HfscQOpt: hfsc,
		}
	}
	qdisc := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: index,
			Handle:  handle,
			Parent:  parent,
			Info:    0,
		},
		Attribute: attrs,
	}
	return qdisc, nil
}

func parseClass(handle, parent uint32, index uint32, cl Class) (tc.Object, error) {
	var attrs tc.Attribute
	switch cl.Type {
	case "hfsc":
		logger.Log("msg", "creating hfsc class")
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
	class := tc.Object{
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
