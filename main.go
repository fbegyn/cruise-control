package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v2"
)

type Config struct {
<<<<<<< HEAD
	Interface string `yaml:"interface"`

	DownloadSpeed string `yaml:"download_speed"`
	UploadSpeed   string `yaml:"upload_speed"`

	Qdiscs  map[string]QdiscConfig `yaml:"qdiscs"`
	Classes map[string]ClassConfig `yaml:"classes"`
}

type QdiscConfig struct {
	Type   string            `yaml:"type"`
	Handle string            `yaml:"handle"`
	Parent string            `yaml:"parent"`
	Specs  map[string]uint32 `yaml:"specs"`
}

type ClassConfig struct {
	Type     string                 `yaml:"type"`
	Classid  string                 `yaml:"classid"`
	Parent   string                 `yaml:"parent"`
	Specs    map[string]interface{} `yaml:"specs"`
	Children map[string]ClassConfig `yaml:"children"`
=======
	Interface string

	DownloadSpeed uint32
	UploadSpeed   uint32

	Qdiscs  map[string]Qdisc
	Classes map[string]Class
	Filters []Filter
}

type Qdisc struct {
	Type   string
	Handle string
	Parent string
	Specs  map[string]uint32
}

type Class struct {
	Type    string
	ClassID string
	Parent  string
	Specs   map[string]interface{}
>>>>>>> config-choice
}

type Filter struct {
	Type string
}

var logger log.Logger

func main() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	f, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		logger.Log("msg", "failed to read in config file")
	}

	conf := Config{}
	yaml.Unmarshal(f, &conf)
	fmt.Println(conf.Classes)

	interf, err := net.InterfaceByName(conf.Interface)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to get interface from name")
	}

	qdMap := make(map[string]tc.Object)
	for qdName, qd := range t.Qdiscs {
		logger.Log("handle", qdName, "type", qd.Type)
		test, _ := parseQdisc(StrHandle(qdName), StrHandle(qd.Parent), uint32(interf.Index), qd)
		qdMap[qdName] = test
		//if err := rtnl.Qdisc().Add(&test); err != nil {
		//	fmt.Fprintf(os.Stderr, "could not assign %s to %s: %v %v\n", handle, t.Interface, err, test.Attribute.FqCodel)
		//	return
		//}
	}

	clMap := make(map[string]tc.Object)
	for clName, cl := range t.Classes {
		logger.Log("classid", clName, "type", cl.Type)
		test, _ := parseClass(StrHandle(cl.ClassID), StrHandle(cl.Parent), uint32(interf.Index), cl)
		clMap[clName] = test
		//if err := rtnl.Qdisc().Add(&test); err != nil {
		//	fmt.Fprintf(os.Stderr, "could not assign %s to %s: %v %v\n", handle, t.Interface, err, test.Attribute.FqCodel)
		//	return
		//}
	}

	fmt.Println(qdMap)
	fmt.Println(clMap)
}

func parseQdisc(handle, parent uint32, index uint32, qd QdiscConfig) (tc.Object, error) {
	logger.Log("msg", "parsing qdisc", "handle", handle, "type", qd.Type)
	var attrs tc.Attribute
	switch qd.Type {
	case "fq_codel":
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

func parseClass(handle, parent uint32, index uint32, cl ClassConfig) (tc.Object, error) {
	logger.Log("msg", "parsing class", "handle", handle, "type", cl.Type)
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
