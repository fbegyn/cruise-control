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
	Type  string
	Specs map[string]uint32
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

	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open rtnetlink socket: %v\n", err)
		return
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "could not close rtnetlink socket: %v\n", err)
		}
	}()

	for handle, qd := range t.Qdiscs {
		logger.Log("handle", handle, "type", qd.Type)
		test, _ := parseQdisc(StrHandle(handle), StrHandle(qd.Parent), uint32(interf.Index), qd)
		if err := rtnl.Qdisc().Add(&test); err != nil {
			fmt.Fprintf(os.Stderr, "could not assign %s to %s: %v %v\n", handle, t.Interface, err, test.Attribute.FqCodel)
			return
		}
	}
}

func parseQdisc(handle, parent uint32, index uint32, qd Qdisc) (tc.Object, error) {
	fmt.Println(handle)
	fmt.Println(parent)
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
