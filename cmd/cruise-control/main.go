package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/florianl/go-tc"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"
)

var (
	logger *slog.Logger
	rtnl   *tc.Tc
)

// Config represents the config in struct shape
type Config struct {
	Addr string
}

func main() {
	flag.Parse()

	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("initializing cruise control")
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		logger.Error("cannot read the config", "err", err)
		os.Exit(1)
	}
	conf := Config{}
	viper.Unmarshal(&conf)

	mux := http.NewServeMux()
	// handle general operations
	mux.HandleFunc("GET /api/v1/tc/{interface}", ObjectListHandler)
	mux.HandleFunc("POST /api/v1/tc/{interface}", ObjectCreateHandler)
	// handle specific operations
	// mux.HandleFunc("GET /api/v1/tc/{interface}/{handle}", ObjectGetHandler)
	mux.HandleFunc("UPDATE /api/v1/tc/{interface}/{handle}", ObjectUpdateHandler)

	logger.Info("starting API server", "addr", conf.Addr)
	if err := http.ListenAndServe(conf.Addr, mux); err != nil {
		logger.Error("cannot start API server", "err", err)
		os.Exit(5)
	}
}

type JsonObject struct {
	Name      string       `json:"name,omitempty"`
	Type      string       `json:"type"`
	Interface string       `json:"interface"`
	Handle    string       `json:"handle"`
	Parent    string       `json:"parent,omitempty"`
	Attr      tc.Attribute `json:"attr"`
}

func ObjectListHandler(w http.ResponseWriter, r *http.Request) {
	// handle general response settings
	w.Header().Set("Content-Type", "application/json")

	// open a go-tc socket
	rtnl, err := CreateTcSocket()
	if err != nil {
		logger.Error("failed to open rtnetlink socket", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to open rtnetlink socket"))
		return
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			logger.Error("failed to close rtnetlink socket", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to close rtnetlink socket"))
			return
		}
		logger.Debug("closed rtnetlink socket")
	}()
	logger.Debug("opened rtnetlink socket")

	JsonArray := []JsonObject{}
	interfaceString := r.PathValue("interface")
	interf, err := net.InterfaceByName(interfaceString)
	if err != nil {
		logger.Error("failed to find interface", "err", err, "interface", interfaceString)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to find interface"))
		return
	}

	qdiscs, err := rtnl.Qdisc().Get()
	if err != nil {
		logger.Error("failed to fetch qdiscs", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	classes, err := rtnl.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: uint32(interf.Index),
	})
	if err != nil {
		logger.Error("failed to fetch classes", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filters, err := rtnl.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: uint32(interf.Index),
	})
	if err != nil {
		logger.Error("failed to fetch filters", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, qd := range qdiscs {
		if qd.Ifindex != uint32(interf.Index) {
			continue
		}
		JsonArray = append(JsonArray, TCToJson(qd, "qdisc"))
	}
	for _, cl := range classes {
		if cl.Ifindex != uint32(interf.Index) {
			continue
		}
		JsonArray = append(JsonArray, TCToJson(cl, "class"))
	}
	for _, fl := range filters {
		if fl.Ifindex != uint32(interf.Index) {
			continue
		}
		JsonArray = append(JsonArray, TCToJson(fl, "filter"))
	}

	bytes, err := json.Marshal(JsonArray)
	if err != nil {
		logger.Error("failed to marshal tc objects into json", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func ObjectCreateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data := []JsonObject{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		logger.Error("failed reading body", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	logger.Info("parsed data packet into TC objects")

	// open a go-tc socket
	rtnl, err := CreateTcSocket()
	if err != nil {
		logger.Error("failed to open rtnetlink socket", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to open rtnetlink socket"))
		return
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			logger.Error("failed to close rtnetlink socket", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to close rtnetlink socket"))
			return
		}
		logger.Debug("closed rtnetlink socket")
	}()
	logger.Debug("opened rtnetlink socket")

	for _, d := range data {
		obj, err := JsonToTC(d)
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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Cruise control updated"))
}

func ObjectUpdateHandler(w http.ResponseWriter, r *http.Request) {
}
