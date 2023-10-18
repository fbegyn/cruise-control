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

	"github.com/florianl/go-tc"
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


	// open a go-tc socket
	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		logger.Error("failed to open rtnetlink socket", "err", err)
		return
	}
	err = rtnl.SetOption(netlink.ExtendedAcknowledge, true)
	if err != nil {
		logger.Error("failed to set extended acknowledge", "err", err)
		return
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
		logger.Error("failed to close rtnetlink socket", "err", err)
		}
	}()
	logger.Info("opened rtnetlink socket")

	for _, d := range data {
		obj, err := JSONToTC(d)
		if err != nil {
			logger.Error("failed to convert JSON into TC obj", "err", err)
			continue
		}
		logger.Info(
			"generated TC object",
			"handle", d.Handle,
			"parent", d.Parent,
			"kind", obj.Attribute.Kind,
			"type", d.Type,
			"interface", d.Interface,
			"ifIndex", obj.Ifindex,
		)

		switch d.Type {
		case "qdisc":
			err = rtnl.Qdisc().Replace(&obj)
			if err != nil {
				logger.Error("failed to replace qdisc", "handle", d.Handle, "err", err)
			}
		case "class":
			err = rtnl.Class().Replace(&obj)
			if err != nil {
				logger.Error("failed to replace class", "handle", d.Handle, "err", err)
			}
		case "filter":
			err = rtnl.Filter().Replace(&obj)
			if err != nil {
				logger.Error("failed to replace filter", "handle", d.Handle, "err", err)
			}
		default:
			logger.Warn("unkown TC object type", "type", d.Type)
		}
	}
}
