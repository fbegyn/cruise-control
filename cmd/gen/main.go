// go generate

package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math"
	"net"

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

type TcConfig struct {
	Qdiscs  map[string]tc.Object
	Classes map[string]tc.Object
	Filters map[string]tc.Object
}

func main() {
	// create a context in which the website can run and add logging
	ctx := opname.With(context.Background(), "main")

	// Enable logging and serve the website
	ln.Log(ctx, ln.Action("rendering_config"))

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")

	if err := viper.ReadInConfig(); err != nil {
		ln.FatalErr(ctx, err)
	}

	conf := Config{}
	viper.Unmarshal(&conf)

	interfacespeed := 1e9 / 8
	conf.DownloadSpeed = conf.DownloadSpeed / 8
	internetspeed := math.Ceil(conf.DownloadSpeed * 0.95)
	reservedspeed := math.Ceil(interfacespeed * 0.2)

	prio1speed := math.Ceil(internetspeed * 0.4)
	prio2speed := math.Ceil(internetspeed * 0.4)
	otherspeed := math.Ceil(internetspeed * 0.2)

	httpspeed := math.Ceil(otherspeed * 0.7)
	browserspeed := math.Ceil(httpspeed * 0.7)
	downloadspeed := math.Ceil(httpspeed * 0.3)

	thrashspeed := math.Ceil(otherspeed * 0.1)
	crewspeed := math.Ceil(otherspeed * 0.2)

	interf, err := net.InterfaceByName(conf.Interface)
	if err != nil {
		ln.FatalErr(ctx, err)
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

	template := TcConfig{
		Qdiscs:  make(map[string]tc.Object),
		Classes: make(map[string]tc.Object),
		Filters: make(map[string]tc.Object),
	}

	template.Qdiscs["root"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x0),
			Parent:  tc.HandleRoot,
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			HfscQOpt: &tc.HfscQOpt{
				DefCls: 3,
			},
			Stab: &tc.Stab{
				Base: &tc.SizeSpec{
					LinkLayer: 1,
					MTU:       1500,
				},
			},
		},
	}
	template.Qdiscs["prio1"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x11, 0x0),
			Parent:  core.BuildHandle(0x1, 0x11),
		},
		Attribute: defaultFqCodel,
	}
	template.Qdiscs["prio2"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x12, 0x0),
			Parent:  core.BuildHandle(0x1, 0x12),
		},
		Attribute: defaultFqCodel,
	}
	template.Qdiscs["browsing"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x31, 0x0),
			Parent:  core.BuildHandle(0x1, 0x31),
		},
		Attribute: defaultFqCodel,
	}
	template.Qdiscs["dowloading"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x32, 0x0),
			Parent:  core.BuildHandle(0x1, 0x32),
		},
		Attribute: defaultFqCodel,
	}
	template.Qdiscs["thrash"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x22, 0x0),
			Parent:  core.BuildHandle(0x1, 0x22),
		},
		Attribute: defaultFqCodel,
	}
	template.Qdiscs["crew"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x23, 0x0),
			Parent:  core.BuildHandle(0x1, 0x23),
		},
		Attribute: defaultFqCodel,
	}
	template.Qdiscs["routing"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x3, 0x0),
			Parent:  core.BuildHandle(0x1, 0x3),
		},
		Attribute: defaultFqCodel,
	}

	interfaceClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x1),
			Parent:  core.BuildHandle(0x1, 0x0),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetSC(interfaceClass.Attribute.Hfsc, uint32(interfacespeed), 0, 0)
	SetUL(interfaceClass.Attribute.Hfsc, uint32(interfacespeed), 0, 0)
	template.Classes["interface"] = interfaceClass

	internetClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x2),
			Parent:  core.BuildHandle(0x1, 0x1),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetSC(internetClass.Attribute.Hfsc, uint32(internetspeed), 0, 0)
	SetUL(interfaceClass.Attribute.Hfsc, uint32(internetspeed), 0, 0)
	template.Classes["internet"] = internetClass

	prio1Class := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x11),
			Parent:  core.BuildHandle(0x1, 0x2),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetSC(prio1Class.Attribute.Hfsc, uint32(prio1speed), 0, 0)
	template.Classes["prio1"] = prio1Class

	prio2Class := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x12),
			Parent:  core.BuildHandle(0x1, 0x2),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetSC(prio2Class.Attribute.Hfsc, uint32(prio2speed), 60000, 0)
	template.Classes["prio2"] = prio2Class

	otherClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x13),
			Parent:  core.BuildHandle(0x1, 0x2),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetLS(otherClass.Attribute.Hfsc, uint32(otherspeed), 100000, 0)
	template.Classes["other"] = otherClass

	httpClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x21),
			Parent:  core.BuildHandle(0x1, 0x13),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetLS(httpClass.Attribute.Hfsc, uint32(httpspeed), 0, 0)
	template.Classes["http"] = httpClass

	browseClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x31),
			Parent:  core.BuildHandle(0x1, 0x21),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetSC(browseClass.Attribute.Hfsc, uint32(browserspeed), 0, 0)
	template.Classes["browse"] = browseClass

	downloadClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x32),
			Parent:  core.BuildHandle(0x1, 0x21),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetLS(downloadClass.Attribute.Hfsc, uint32(downloadspeed), 10000, 0)
	template.Classes["download"] = downloadClass

	crewClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x23),
			Parent:  core.BuildHandle(0x1, 0x13),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetLS(crewClass.Attribute.Hfsc, uint32(crewspeed), 0, 0)
	template.Classes["crew"] = crewClass

	thrashClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x23),
			Parent:  core.BuildHandle(0x1, 0x13),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetLS(thrashClass.Attribute.Hfsc, uint32(thrashspeed), 50000, 0)
	template.Classes["thrash"] = thrashClass

	reservedClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0x1, 0x3),
			Parent:  core.BuildHandle(0x1, 0x1),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{},
				Usc: &tc.ServiceCurve{},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	SetLS(reservedClass.Attribute.Hfsc, uint32(reservedspeed), 0, 0)
	template.Classes["reserved"] = reservedClass

	prio1Handle := template.Classes["prio1"].Msg.Handle
	template.Filters["prio1"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Parent:  core.BuildHandle(0x1, 0x0),
		},
		Attribute: tc.Attribute{
			Kind: "u32",
			U32: &tc.U32{
				ClassID: &prio1Handle,
				Mark: &tc.U32Mark{
					Val:  0x1,
					Mask: 0xf,
				},
			},
		},
	}
	prio2Handle := template.Classes["prio2"].Msg.Handle
	template.Filters["prio2"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Parent:  core.BuildHandle(0x1, 0x0),
		},
		Attribute: tc.Attribute{
			Kind: "u32",
			U32: &tc.U32{
				ClassID: &prio2Handle,
				Mark: &tc.U32Mark{
					Val:  0x2,
					Mask: 0xf,
				},
			},
		},
	}
	browseHandle := template.Classes["browse"].Msg.Handle
	template.Filters["other"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Parent:  core.BuildHandle(0x1, 0x0),
		},
		Attribute: tc.Attribute{
			Kind: "u32",
			U32: &tc.U32{
				ClassID: &browseHandle,
				Mark: &tc.U32Mark{
					Val:  0x3,
					Mask: 0xf,
				},
			},
		},
	}
	downloadHandle := template.Classes["download"].Msg.Handle
	template.Filters["download"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Parent:  core.BuildHandle(0x1, 0x0),
		},
		Attribute: tc.Attribute{
			Kind: "u32",
			U32: &tc.U32{
				ClassID: &downloadHandle,
				Mark: &tc.U32Mark{
					Val:  0x4,
					Mask: 0xf,
				},
			},
		},
	}
	crewHandle := template.Classes["crew"].Msg.Handle
	template.Filters["crew"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Parent:  core.BuildHandle(0x1, 0x0),
		},
		Attribute: tc.Attribute{
			Kind: "u32",
			U32: &tc.U32{
				ClassID: &crewHandle,
				Mark: &tc.U32Mark{
					Val:  0x5,
					Mask: 0xf,
				},
			},
		},
	}
	thrashHandle := template.Classes["thrash"].Msg.Handle
	template.Filters["thrash"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Parent:  core.BuildHandle(0x1, 0x0),
		},
		Attribute: tc.Attribute{
			Kind: "u32",
			U32: &tc.U32{
				ClassID: &thrashHandle,
				Mark: &tc.U32Mark{
					Val:  0x6,
					Mask: 0xf,
				},
			},
		},
	}
	reservedHandle := template.Classes["reserved"].Msg.Handle
	template.Filters["reserved"] = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Parent:  core.BuildHandle(0x1, 0x0),
		},
		Attribute: tc.Attribute{
			Kind: "fw",
			Fw: &tc.Fw{
				ClassID: &reservedHandle,
				InDev:   &conf.Interface,
			},
		},
	}

	// need to figure the inital filter for interface traffic and routing
	// template.Filters["arp"] = tc.Object{
	// 	Msg: tc.Msg{
	// 		Family:  unix.AF_UNSPEC,
	// 		Ifindex: uint32(interf.Index),
	// 		Parent:  core.BuildHandle(1, 0),
	// 	},
	// 	Attribute: tc.Attribute{
	// 		Kind: "fw",
	// 		Fw: &tc.Fw{
	// 			ClassID: &reservedHandle,
	// 			InDev:   &conf.Interface,
	// 		},
	// 	},
	// }

	// render the config to the JSON file
	b, err := json.Marshal(template)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	err = ioutil.WriteFile("./highway.json", b, 0644)
	if err != nil {
		ln.FatalErr(ctx, err)
	}
}
