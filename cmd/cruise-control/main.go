package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"strconv"

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
	Port      int

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
	flag.Parse()

	ctx := opname.With(context.Background(), "main")
	ln.Log(ctx, ln.Action("initializing cruise control"))
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		ln.FatalErr(ctx, err)
	}
	conf := Config{}
	viper.Unmarshal(&conf)

	http.HandleFunc("/tc/apply", TCApplyHandler)
	ln.Log(ctx, ln.Info("starting API on 0.0.0.0:%d", conf.Port))
	ln.FatalErr(ctx, http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil))
}

func TCApplyHandler(w http.ResponseWriter, r *http.Request) {
	ctx := opname.With(context.Background(), "TCApplyHandler")
	devName := r.URL.Query().Get("interface")
	speed, err := strconv.Atoi(r.URL.Query().Get("up"))
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	ln.Log(ctx, ln.Info("interface: %s - speed: %d Mbps", devName, speed))

	interf, err := net.InterfaceByName(devName)
	tcConf := createQoSSimple(ctx, *interf, 1e9, speed)

	// construct the TC nodes from the config file
	var nodes []*Node
	for _, class := range tcConf.Classes {
		n := NewNodeWithObject("class", class)
		nodes = append(nodes, n)
	}
	for _, qdisc := range tcConf.Qdiscs {
		n := NewNodeWithObject("qdisc", qdisc)
		nodes = append(nodes, n)
	}

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
	nodes = tree.ComposeChildren(nodes)
	if len(nodes) == 0 {
		ln.Log(ctx, ln.Info("all TC nodes parsed, tree constructed"))
	} else {
		ln.Log(ctx, ln.Info("there are leftover TC nodes: %d nodes left", len(nodes)))
	}

	// open a go-tc socket
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

	// get the system tree and compare it to the current config. If there is a difference, we should
	// reapply the tree so the config is matched
	ln.Log(ctx, ln.Action("Fetching current TC state"))
	systemNodes, systemFilters := GetInterfaceNodes(rtnl, uint32(interf.Index))
	systemTree, index := FindRootNode(systemNodes)
	systemNodes = append(systemNodes[:index], systemNodes[index+1:]...)
	systemTree.ComposeChildren(systemNodes)

	// check if the system is up to date or not
	if !systemTree.CompareTree(*tree) {
		ln.Log(ctx, ln.Info("updating the current interfaces qdiscs and classes"))
		tree.ApplyNode(rtnl)
	} else {
		ln.Log(ctx, ln.Info("current interface is already up to date with the qdiscs and classes"))
	}

	test := make(map[uint32]struct{})
	for _, t := range systemFilters {
		if _, ok := test[t.Object.Handle]; ok {
			continue
		}
		test[t.Object.Handle] = struct{}{}
	}

	ln.Log(ctx, ln.Action("Applying filters"))
	for _, filt := range filters {
		if _, k := test[filt.Object.Handle]; k {
			fmt.Println("fitler found")
		}
		filt.ApplyNode(rtnl)
	}

}
