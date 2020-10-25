package main

import (
	"fmt"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
)

type Config struct {
	Interface string

	DownloadSpeed float64
	UploadSpeed   float64

	Qdiscs  map[string]QdiscConfig
	Classes map[string]ClassConfig
	Filters map[string]FilterConfig
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

type FilterConfig struct {
	Type     string
	FilterID string
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

	// determine the interface for cruise control to run on
	//interf, err := net.InterfaceByName(conf.Interface)
	//if err != nil {
	//	logger.Log("level", "ERROR", "msg", "failed to get interface from name")
	//}

	/* we first need to parse all the attributes of the components into a map of
	* attributes and identifiers. This is required because the parent of each
	* component can be any of the other types. The choice is:
	* 1. Have the requirement of having unique names for each object
	* 2. Not have unique names but then seperate maps need to be searched through
	* to find the parents, complicating things unnecesarily since the same
	* identifiers can be found */

	h1, err := parseHandles(conf)
	if err != nil {
		os.Exit(8)
	}
	fmt.Println(h1)

	h, err := parseParents(h1, conf)
	if err != nil {
		os.Exit(9)
	}
	fmt.Println(h)

	os.Exit(10)

	// compose TC objects into maps of each type
	//qdMap := composeQdiscs(conf.Qdiscs)
	//clMap := composeClasses(conf.Classes)
	//flMap := composeFilters(conf.Filters)

	// compose the parents of each object in the tree
	//for k, v := range conf.Classes {
	//	p := v.Parent
	//	if _, ok := clMap[p]; !ok {
	//		continue
	//	}
	//	clMap[k] = clMap[p].Msg.Handle
	//}
	//for k, v := range conf.Qdiscs {
	//	p := v.Parent
	//	if _, ok := clMap[p]; !ok {
	//		continue
	//	}
	//	qdMap[k].Msg.Parent = clMap[p].Msg.Handle
	//}

	//// constrcut tc objects into an array
	//var nodes []*Node
	//for k, v := range qdMap {
	//	n := NewNodeWithObject(k, "qdisc", *v)
	//	nodes = append(nodes, n)
	//}
	//for k, v := range clMap {
	//	n := NewNodeWithObject(k, "class", *v)
	//	nodes = append(nodes, n)
	//}
	//for k, v := range flMap {
	//	n := NewNodeWithObject(k, "filter", *v)
	//	nodes = append(nodes, n)
	//}

	//// construct the TC tree
	//tree, index := FindRootNode(nodes)
	//nodes = append(nodes[:index], nodes[index+1:]...)
	//leftover := tree.ComposeChildren(nodes)
	//nodes = leftover

	//rtnl, err := tc.Open(&tc.Config{})
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "could not open rtnetlink socket: %v\n", err)
	//	return
	//}
	//defer func() {
	//	if err := rtnl.Close(); err != nil {
	//		fmt.Fprintf(os.Stderr, "could not close rtnetlink socket: %v\n", err)
	//	}
	//}()

	//if len(nodes) == 0 {
	//	logger.Log("level", "INFO", "msg", "all nodes parsed, no nodes left over")
	//} else {
	//	logger.Log("level", "INFO", "msg", "some nodes left over")
	//}

	//systemNodes := GetInterfaceNodes(rtnl, uint32(interf.Index))
	//systemTree := ComposeTree(systemNodes)

	//if !systemTree.CompareTree(tree) {
	//	logger.Log("level", "INFO", "msg", "applying new config")
	//	systemTree.UpdateTree(tree, rtnl)
	//} else {
	//	logger.Log("level", "INFO", "msg", "config up to date")
	//}
}
