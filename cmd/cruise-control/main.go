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

	DownloadSpeed uint32
	UploadSpeed   uint32

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

	qdMap := make(map[string]tc.Object)
	for qdName, qd := range conf.Qdiscs {
		logger.Log("handle", qdName, "type", qd.Type)
		qdisc, err := parseQdisc(StrHandle(qdName), StrHandle(qd.Parent), uint32(interf.Index), qd)
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

	clMap := make(map[string]tc.Object)
	for clName, cl := range conf.Classes {
		logger.Log("classid", clName, "type", cl.Type)
		class, err := parseClass(StrHandle(cl.ClassID), StrHandle(cl.Parent), uint32(interf.Index), cl)
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
}
