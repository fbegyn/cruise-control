package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/mdlayher/netlink"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"
	"within.website/ln"
	"within.website/ln/opname"
)

// Config represents the config in struct shape
//
//go:generate go run ../gen/main.go ../gen/helpers.go
type Config struct {
	Addr string
}

var (
	logger *slog.Logger
)

func main() {
	flag.Parse()

	ctx := opname.With(context.Background(), "main")
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("initializing cruise control")
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		ln.FatalErr(ctx, err)
	}
	conf := Config{}
	viper.Unmarshal(&conf)

	mux := http.NewServeMux()

	// http.HandleFunc("/api/tc/apply", TCApplyHandler)
	// http.HandleFunc("/api/tc/create", TCCreateHandler)
	mux.HandleFunc("/api/tc/objects", ObjectCreateHandler)
	// http.HandleFunc("/api/tc/handles/<handle>", HandleGetHandler)
	// http.HandleFunc("/api/tc/handles/<handle>", HandlePutHandler)
	logger.Info("starting API server", "addr", conf.Addr)
	if err := http.ListenAndServe(conf.Addr, mux); err != nil {
		logger.Error("cannot start API server", "err", err)
	}
}

type JSONObject struct {
	Name      string       `json:"name,omitempty"`
	Type      string       `json:"type"`
	Interface string       `json:"interface"`
	Handle    string       `json:"handle"`
	Parent    string       `json:"parent,omitempty"`
	Attr      tc.Attribute `json:"attr"`
}

func JSONToTC(json JSONObject) (tc.Object, error) {
	handle, err := StrHandle(json.Handle)
	if err != nil {
		logger.Error("failed to parse handle", "handle", json.Handle, "err", err)
		return tc.Object{}, fmt.Errorf("failed to parse handle")
	}
	parent, err := StrHandle(json.Parent)
	if err != nil {
		logger.Error("failed to parse parent", "parent", json.Handle, "err", err)
		return tc.Object{}, fmt.Errorf("failed to parse parent")
	}
	interf, err := net.InterfaceByName(json.Interface)
	if err != nil {
		logger.Error("failed to lookup interface", "interface", json.Interface, "err", err)
		return tc.Object{}, fmt.Errorf("failed to lookup interface")
	}
	return tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  handle,
			Parent:  parent,
		},
		Attribute: json.Attr,
	}, nil
}

func ObjectCreateHandler(w http.ResponseWriter, r *http.Request) {
	data := []JSONObject{}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		logger.Error("failed reading body", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	logger.Info("parsed data packet into TC objects")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("Cruise control updated"))

	fmt.Println(data)
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

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/text")
	w.Write([]byte("Cruise control updated"))
}
