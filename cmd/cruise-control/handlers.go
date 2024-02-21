package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"golang.org/x/sys/unix"
)

type JsonObject struct {
	Name      string       `json:"name,omitempty"`
	Type      string       `json:"type"`
	Interface string       `json:"interface"`
	Handle    string       `json:"handle"`
	Parent    string       `json:"parent,omitempty"`
	Attr      tc.Attribute `json:"attr"`
}

func TCToJson(obj tc.Object, tcType string) JsonObject {
	maj, min := core.SplitHandle(obj.Handle)
	jsonObj := JsonObject{
		Type:      tcType,
		Interface: fmt.Sprintf("%d", obj.Ifindex),
		Handle:    fmt.Sprintf("%d:%d", maj, min),
		Attr:      obj.Attribute,
	}
	if obj.Parent != 0 {
		maj, min = core.SplitHandle(obj.Parent)
		jsonObj.Parent = fmt.Sprintf("%d:%d", maj, min)
	}
	return jsonObj
}

func JsonToTC(payload JsonObject) (tc.Object, error) {
	interf, err := net.InterfaceByName(payload.Interface)
	if err != nil {
		return tc.Object{}, err
	}
	handle, err := StrHandle(payload.Handle)
	parent, err := StrHandle(payload.Parent)

	obj := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  handle,
			Parent:  parent,
		},
		Attribute: payload.Attr,
	}
	return obj, nil
}

func ObjectListHandler(w http.ResponseWriter, r *http.Request) {
	// handle general response settings
	w.Header().Set("Content-Type", "application/json")

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
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("failed to close rtnetlink socket"))
				return
			}
		case "class":
			err = rtnl.Class().Replace(&obj)
			if err != nil {
				logger.Error("failed to replace class", "handle", d.Handle, "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("failed to close rtnetlink socket"))
				return
			}
		case "filter":
			err = rtnl.Filter().Replace(&obj)
			if err != nil {
				logger.Error("failed to replace filter", "handle", d.Handle, "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("failed to close rtnetlink socket"))
				return
			}
		default:
			logger.Warn("unkown TC object type", "type", d.Type)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("unkown TC object type"))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Cruise control updated"))
}

func ObjectUpdateHandler(w http.ResponseWriter, r *http.Request) {
	// handle general response settings
	w.Header().Set("Content-Type", "application/json")

	JsonArray := []JsonObject{}
	interfaceString := r.PathValue("interface")
	interf, err := net.InterfaceByName(interfaceString)
	if err != nil {
		logger.Error("failed to find interface", "err", err, "interface", interfaceString)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to find interface"))
		return
	}
	handleString := r.PathValue("handle")
	handleParse, err := strconv.ParseUint(handleString, 10, 32)
	if err != nil {
		logger.Error("failed to parse handle", "err", err, "handle", handleString)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to parse handle"))
		return
	}
	handle := uint32(handleParse)
	_ = handle
	logger.Info("handle", handleString, "json", JsonArray, "interface", interf)
	return
}

func ObjectGetHandler(w http.ResponseWriter, r *http.Request) {
	// handle general response settings
	w.Header().Set("Content-Type", "application/json")

	JsonArray := []JsonObject{}
	interfaceString := r.PathValue("interface")
	interf, err := net.InterfaceByName(interfaceString)
	if err != nil {
		logger.Error("failed to find interface", "err", err, "interface", interfaceString)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to find interface"))
		return
	}
	handleString := r.PathValue("handle")
	handleParse, err := strconv.ParseUint(handleString, 10, 32)
	if err != nil {
		logger.Error("failed to parse handle", "err", err, "handle", handleString)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to parse handle"))
		return
	}
	handle := uint32(handleParse)

	qdiscs, err := rtnl.Qdisc().Get()
	if err != nil {
		logger.Error("failed to fetch qdiscs", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	classes, err := rtnl.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: uint32(interf.Index),
		Handle:  handle,
	})
	if err != nil {
		logger.Error("failed to fetch classes", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filters, err := rtnl.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: uint32(interf.Index),
		Handle:  handle,
	})
	if err != nil {
		logger.Error("failed to fetch filters", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, qd := range qdiscs {
		if qd.Ifindex != uint32(interf.Index) || qd.Handle != handle {
			continue
		}
		JsonArray = append(JsonArray, TCToJson(qd, "qdisc"))
	}
	for _, cl := range classes {
		if cl.Ifindex != uint32(interf.Index) || cl.Handle != handle {
			continue
		}
		JsonArray = append(JsonArray, TCToJson(cl, "class"))
	}
	for _, fl := range filters {
		if fl.Ifindex != uint32(interf.Index) || fl.Handle != handle {
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
