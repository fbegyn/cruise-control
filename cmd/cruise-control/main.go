package main

import (
	"fmt"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
)

// Config represents the config in struct shape
type Config struct {
	Interface string

	DownloadSpeed float64
	UploadSpeed   float64

	Qdiscs  map[string]QdiscConfig
	Classes map[string]ClassConfig
	Filters map[string]FilterConfig
}

// QdiscConfig represents the Qdisc config
type QdiscConfig struct {
	Type   string
	Handle string
	Parent string
	Specs  map[string]uint32
}

// ClassConfig represents that Class config
type ClassConfig struct {
	Type    string
	ClassID string
	Parent  string
	Specs   map[string]interface{}
}

// FilterConfig represents a TC filter config in struct
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
	interf, err := net.InterfaceByName(conf.Interface)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to get interface from name")
	}

	// compose a map of all handles (also classIDs) to easily lookup the qdisc or
	// class when needed
	handleMap, err := parseHandles(conf)
	if err != nil {
		logger.Log("error", err)
		os.Exit(8)
	}
	// compose a map of all parent of the object to easily look them up later
	parentMap, err := parseParents(handleMap, conf)
	if err != nil {
		logger.Log("error", err)
		os.Exit(9)
	}

	// compose TC objects into maps of each type. we could also create a single
	// map of tc.Objects but that what probably include som wizardly or adding a
	// field to the objects configs to determine the type.
	qdMap, _ := composeQdiscs(handleMap, parentMap, conf.Qdiscs, interf)
	clMap, _ := composeClasses(handleMap, parentMap, conf.Classes, interf)
	//flMap, _ := composeFilters(handleMap, parentMap, conf.Filters, interf)

	// construct tc objects into an array
	var nodes []*Node
	for k, v := range qdMap {
		n := NewNodeWithObject(k, "qdisc", *v)
		nodes = append(nodes, n)
	}
	for k, v := range clMap {
		n := NewNodeWithObject(k, "class", *v)
		nodes = append(nodes, n)
	}

	// construct the TC tree
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

	if len(nodes) == 0 {
		logger.Log("level", "INFO", "msg", "all nodes parsed, no nodes left over","tree","config")
	} else {
		logger.Log("level", "INFO", "msg", "some nodes left over","tree","config")
	}

	systemNodes := GetInterfaceNodes(rtnl, uint32(interf.Index), handleMap)
	if len(systemNodes) > 0 {
		systemTree, index := FindRootNode(systemNodes)
		systemNodes = append(systemNodes[:index], systemNodes[index+1:]...)
		systemNodes = systemTree.ComposeChildren(systemNodes)
		if !systemTree.CompareTree(tree){
		    logger.Log("level","INFO","msg","updating config")
		    systemTree.DeleteNode(rtnl)
		    tree.ApplyNode(rtnl)
		} else {
		    logger.Log("level","INFO","msg","system already up to date")
		}
	} else {
		logger.Log("level","INFO","msg","no config found, applying config file")
		tree.ApplyNode(rtnl)
	}

}
