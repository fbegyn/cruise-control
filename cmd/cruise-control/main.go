package main

import (
	"fmt"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
)

type Config struct {
	Interface string

	DownloadSpeed float64
	UploadSpeed   float64

	Qdiscs  map[string]QdiscConfig
	Classes map[string]ClassConfig
	Filters []Filter
}

type QdiscConfig struct {
	Type   string
	Handle string
	Parent string
	Specs  map[string]uint32
}

type ClassConfig struct {
	Type    string
	ClassID string
	Parent  string
	Specs   map[string]interface{}
}

type Filter struct {
	Type string
}

var logger log.Logger

func main() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")

	if err := viper.ReadInConfig(); err != nil {
		logger.Log("level", "ERROR", "msg", "failed to read config file", "error", err)
	}

	conf := Config{}
	viper.Unmarshal(&conf)
	fmt.Println(conf)

	// determine the interface for cruise control to run on
	interf, err := net.InterfaceByName(conf.Interface)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to get interface from name")
	}

	qdMap := make(map[string]*tc.Object)
	for qdName, qd := range conf.Qdiscs {
		logger.Log("level", "INFO", "msg", "parsing qdisc", "name", qdName, "handle", qd.Handle, "type", qd.Type)
		qdisc, err := parseQdisc(StrHandle(qd.Handle), StrHandle(qd.Parent), uint32(interf.Index), qd)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
		} else {
			logger.Log("level", "INFO", "msg", "qdisc parsed and adding to map")
			qdMap[qdName] = qdisc
		}
		//if err := rtnl.Qdisc().Add(&test); err != nil {
		//	fmt.Fprintf(os.Stderr, "could not assign %s to %s: %v %v\n", handle, conf.Interface, err, test.Attribute.FqCodel)
		//	return
		//}
	}

	clMap := make(map[string]*tc.Object)
	for clName, cl := range conf.Classes {
		logger.Log("level", "INFO", "msg", "parsing clas", "name", clName, "handle", cl.ClassID, "type", cl.Type)
		class, err := parseClass(StrHandle(cl.ClassID), StrHandle(cl.Parent), uint32(interf.Index), conf.DownloadSpeed, cl)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse qdisc")
		} else {
			logger.Log("level", "INFO", "msg", "qdisc parsed and adding to map")
			clMap[clName] = class
		}
		//if err := rtnl.Qdisc().Add(&test); err != nil {
		//	fmt.Fprintf(os.Stderr, "could not assign %s to %s: %v %v\n", handle, t.Interface, err, test.Attribute.FqCodel)
		//	return
		//}
	}

	for k, v := range conf.Classes {
		p := v.Parent
		if _, ok := clMap[p]; !ok {
			continue
		}
		clMap[k].Msg.Parent = clMap[p].Msg.Handle
	}

	for k, v := range conf.Qdiscs {
		p := v.Parent
		if _, ok := clMap[p]; !ok {
			continue
		}
		qdMap[k].Msg.Parent = clMap[p].Msg.Handle
	}

	var nodes []*Node
	for _, v := range qdMap {
		n := NewNode(v, "qdisc")
		nodes = append(nodes, n)
	}
	for _, v := range clMap {
		n := NewNode(v, "class")
		nodes = append(nodes, n)
	}

	tree, index := FindRootNode(nodes)
	nodes = append(nodes[:index], nodes[index+1:]...)
	leftover := tree.ComposeChildren(nodes)
	nodes = leftover

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

	tree.ApplyNode(rtnl)
}
