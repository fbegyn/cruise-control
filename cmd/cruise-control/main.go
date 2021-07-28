package main

import (
	"context"
	"net"

	"github.com/florianl/go-tc"
	"github.com/mdlayher/netlink"
	"github.com/spf13/viper"
	"within.website/ln"
	"within.website/ln/opname"
)

//go:generate go run ../gen/main.go ../gen/helpers.go

// Config represents the config in struct shape
type Config struct {
	Interface string

	DownloadSpeed float64
	UploadSpeed   float64

	TrafficFile string
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
	Parent   string
	Specs    map[string]interface{}
}

func main() {
	// create a context in which the website can run and add logging
	ctx := opname.With(context.Background(), "main")

	// Enable logging and serve the website
	ln.Log(ctx, ln.Action("initializing cruise control"))

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")

	if err := viper.ReadInConfig(); err != nil {
		ln.FatalErr(ctx, err)
	}

	conf := Config{}
	viper.Unmarshal(&conf)

	// determine the interface for cruise control to run on
	interf, err := net.InterfaceByName(conf.Interface)
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	_ = interf

	ln.Log(ctx, ln.Action("determining cruise speed"))
	tcConf, err := parseTrafficFile(conf.TrafficFile)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	// update the config to match the interface we're editiong
	// if the config is rendered somewhere else, this is needed
	// to ensure that we don't modify the wrong interface if the
	// index exists
	tcConf.updateInterface(*interf)

	// construct the TC nodes from the config file
	var nodes []*Node
	// first the classes
	for _, class := range tcConf.Classes {
		n := NewNodeWithObject("class", class)
		nodes = append(nodes, n)
	}
	// then the qdiscs
	for _, qdisc := range tcConf.Qdiscs {
		n := NewNodeWithObject("qdisc", qdisc)
		nodes = append(nodes, n)
	}

	// we load the filters as a seperate TC node "pool"
	var filters []*Node
	for _, filter := range tcConf.Filters {
		n := NewNodeWithObject("filter", filter)
		filters = append(filters, n)
	}

	// construct the TC tree
	tree, index := FindRootNode(nodes)
	if tree == nil {
		ln.FatalErr(ctx, err)
	}
	nodes = append(nodes[:index], nodes[index+1:]...)
	leftover := tree.ComposeChildren(nodes)
	nodes = leftover

	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		ln.FatalErr(ctx, err)
		return
	}
	err = rtnl.SetOption(netlink.ExtendedAcknowledge, true)
	if err != nil {
		ln.FatalErr(ctx, err)
		return
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			ln.FatalErr(ctx, err)
		}
	}()

	if len(nodes) == 0 {
		ln.Log(ctx, ln.Info("all TC nodes parsed, no TC nodes left over"))
	} else {
		ln.Log(ctx, ln.Info("there are leftover TC nodes: %d nodes left", len(nodes)))
	}

	ln.Log(ctx, ln.Action("Fetching current TC state"))
	systemNodes := GetInterfaceNodes(rtnl, uint32(interf.Index))
	ln.Log(ctx, ln.Action("Applying qdiscs and classes"))
	if len(systemNodes) > 0 {
		systemTree, index := FindRootNode(systemNodes)
		systemNodes = append(systemNodes[:index], systemNodes[index+1:]...)
		systemTree.ComposeChildren(systemNodes)
		if !systemTree.CompareTree(*tree) {
			ln.Log(ctx, ln.Info("updating the current interfaces TC config, please wait ..."))
			systemTree.DeleteNode(rtnl)
			tree.ApplyNode(rtnl)
		} else {
			ln.Log(ctx, ln.Info("current TC config is already up to date"))
		}
	} else {
		ln.Log(ctx, ln.Info("current interface does not have TC data, applying config ..."))
		tree.ApplyNode(rtnl)
	}

	ln.Log(ctx, ln.Action("Applying filters"))
	for _, filt := range filters {
		filt.ApplyNode(rtnl)
	}
}
