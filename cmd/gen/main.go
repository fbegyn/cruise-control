// go generate

package main

import (
	"context"
	"encoding/json"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/spf13/viper"
	"within.website/ln"
	"within.website/ln/opname"

	"golang.org/x/sys/unix"
)

// Config represents the config in struct shape
type Config struct {
	Interface string

	DownloadSpeed float64
	UploadSpeed   float64
}

func main() {

	// create a context in which the website can run and add logging
	ctx := opname.With(context.Background(), "main")

	// Enable logging and serve the website
	ln.Log(ctx, ln.Action("starting_application"))

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")

	if err := viper.ReadInConfig(); err != nil {
		ln.FatalErr(ctx, err)
	}

	conf := Config{}
	viper.Unmarshal(&conf)

	interfacespeed := 1e9
	internetspeed := conf.DownloadSpeed * 0.95
	// routingspeed := interfacespeed * 0.2

	prio1speed := internetspeed * 0.4
	// prio2speed := internetspeed * 0.4
	// otherspeed := internetspeed * 0.2

	// httpspeed := otherspeed * 0.7
	// // browserspeed := httpspeed * 0.7
	// downloadspeed := httpspeed * 0.3

	// thrashspeed := otherspeed * 0.1
	// crewspeed := otherspeed * 0.2

	interf, err := net.InterfaceByName(conf.Interface)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	defaultHfsc := tc.Attribute{
		Kind: "hfsc",
		Hfsc: &tc.Hfsc{
			Rsc: &tc.ServiceCurve{},
			Usc: &tc.ServiceCurve{},
			Fsc: &tc.ServiceCurve{},
		},
	}

	ecn := uint32(0)
	limit := uint32(1200)
	flows := uint32(65535)
	target := uint32(5000)

	defaultFqCodel := tc.Attribute{
		Kind: "fq_codel",
		FqCodel: &tc.FqCodel{
			ECN:    &ecn,
			Limit:  &limit,
			Flows:  &flows,
			Target: &target,
		},
	}

	rootQdisc := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(1, 0),
			Parent:  tc.HandleRoot,
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			HfscQOpt: &tc.HfscQOpt{
				DefCls: 3,
			},
			Stab: &tc.Stab{
				Base: &tc.SizeSpec{
					MTU: 1500,
				},
			},
		},
	}

	interfaceClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(1, 1),
			Parent:  core.BuildHandle(1, 0),
		},
		Attribute: defaultHfsc,
	}
	SetSC(interfaceClass.Attribute.Hfsc, uint32(interfacespeed), 0, 0)
	SetUL(interfaceClass.Attribute.Hfsc, uint32(interfacespeed), 0, 0)

	internetClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(1, 11),
			Parent:  core.BuildHandle(1, 2),
		},
		Attribute: defaultHfsc,
	}
	SetSC(internetClass.Attribute.Hfsc, uint32(internetspeed), 0, 0)
	SetUL(interfaceClass.Attribute.Hfsc, uint32(internetspeed), 0, 0)

	prio1Class := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(1, 2),
			Parent:  core.BuildHandle(1, 1),
		},
		Attribute: defaultHfsc,
	}
	SetSC(prio1Class.Attribute.Hfsc, uint32(prio1speed), 0, 0)

	prio1Qdisc := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(11, 0),
			Parent:  core.BuildHandle(1, 11),
		},
		Attribute: defaultFqCodel,
	}

	qdiscs := []*tc.Object{}
	qdiscs = append(qdiscs, &rootQdisc)
	qdiscs = append(qdiscs, &prio1Qdisc)

	b, _ := json.Marshal(qdiscs)
	os.Stdout.Write(b)
}
