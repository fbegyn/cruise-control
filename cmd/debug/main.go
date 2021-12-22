package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"

	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

const nlafNested = (1 << 15)

func main() {
	interf, err := net.InterfaceByName("test-01")
	if err != nil {
		panic(err)
	}

	// open up tc socker for the test interface
	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		panic(err)
	}
	err = rtnl.SetOption(netlink.ExtendedAcknowledge, true)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			panic(err)
		}
	}()

	// add the HFSC qdisc to the root ffff:0000
	testQdisc := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0xFFFF, 0x0000),
			Parent:  tc.HandleRoot,
			Info:    0,
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			HfscQOpt: &tc.HfscQOpt{
				DefCls: 1,
			},
		},
	}
	err = rtnl.Qdisc().Replace(&testQdisc)
	if err != nil {
		panic(err)
	}

	// provide a class for HFSC classid: ffff:0001
	internetClass := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Handle:  core.BuildHandle(0xFFFF, 0x0001),
			Parent:  core.BuildHandle(0xFFFF, 0x0000),
		},
		Attribute: tc.Attribute{
			Kind: "hfsc",
			Hfsc: &tc.Hfsc{
				Rsc: &tc.ServiceCurve{100e6, 0, 0},
				Usc: &tc.ServiceCurve{100e6, 0, 0},
				Fsc: &tc.ServiceCurve{},
			},
		},
	}
	err = rtnl.Class().Replace(&internetClass)
	if err != nil {
		panic(err)
	}

	// set a filter for match traffic to the internetclass
	testFilter := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(interf.Index),
			Parent:  core.BuildHandle(0xFFFF, 0x0000),
			Info:    768,
		},
		Attribute: tc.Attribute{
			Kind: "u32",
			U32: &tc.U32{
				ClassID: &internetClass.Msg.Handle,
				Sel:     &tc.U32Sel{},
				Mark: &tc.U32Mark{
					Val:  0x1,
					Mask: 0xf,
				},
			},
		},
	}

	err = rtnl.Filter().Replace(&testFilter)
	if err != nil {
		panic(err)
	}
}

// notes:
// @florian: and app sets also TCA_U32_UNSPEC (0) which should be TCA_U32_CLASSID (1) instead, I guess
//           setting xx_UNSPEC always returns an invalid argument

func decodeNetlink(data []byte, lvl int) error {
	ad, err := netlink.NewAttributeDecoder(data)
	if err != nil {
		return err
	}
	for ad.Next() {
		space := strings.Repeat(" ", lvl)
		if ad.Type()&nlafNested == nlafNested {
			// The data is formatted in the following structure
			// <number> - <byte data>
			// <number>: types from the enum struct
			// <byte data>: byte values for this struct
			fmt.Printf("%s %4d | %d\n", space, ad.Type(), ad.Type()^nlafNested)
			decodeNetlink(ad.Bytes(), lvl+3)
			continue
		}
		fmt.Printf("%s %4d\t%#v\n", space, ad.Type(), ad.Bytes())
	}
	return nil
}

func debug() {
	// Here we need to define the netlink data from TCA_options. Later it is then possible
	// to lookup the output in the linux kernel docs structs for the things.
	tests := map[string][]byte{
		// sendmsg(3, {msg_name={sa_family=AF_NETLINK, nl_pid=0, nl_groups=00000000}, msg_namelen=12, msg_iov=[{iov_base=[{nlmsg_len=92, nlmsg_type=RTM_NEWTFILTER, nlmsg_flags=NLM_F_REQUEST|NLM_F_ACK|NLM_F_EXCL|NLM_F_CREATE, nlmsg_seq=1640201947, nlmsg_pid=0}, {tcm_family=AF_UNSPEC, tcm_ifindex=if_nametoindex("test-01"), tcm_handle=0, tcm_parent=65536, tcm_info=768}, [[{nla_len=8, nla_type=TCA_KIND}, "u32"], [{nla_len=48, nla_type=TCA_OPTIONS}, ",0x10,0x00,0x0a,0x00,0x01,0x00,0x00,0x00,0x0f,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x08,0x00,0x01,0x00,0x11,0x00,0x01,0x00,0x14,0x00,0x05,0x00,0x01,0x00,0x00,0x00"...]]], iov_len=92}], msg_iovlen=1, msg_controllen=0, msg_flags=0}, 0) = 92
		"iproute": []byte{0x10, 0x00, 0x0a, 0x00, 0x01, 0x00, 0x00, 0x00, 0x0f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x00, 0x11, 0x00, 0x01, 0x00, 0x14, 0x00, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00},
		// sendmsg(3, {msg_name={sa_family=AF_NETLINK, nl_pid=0, nl_groups=00000000}, msg_namelen=12, msg_iov=[{iov_base={{nlmsg_len=72, nlmsg_type=RTM_NEWTFILTER, nlmsg_flags=NLM_F_REQUEST|NLM_F_ACK|NLM_F_CREATE, nlmsg_seq=563752620, nlmsg_pid=28466}, {tcm_family=AF_UNSPEC, tcm_ifindex=if_nametoindex("test-01"), tcm_handle=0, tcm_parent=65536, tcm_info=0}, [{{nla_len=8, nla_type=TCA_KIND}, "u32"}, {{nla_len=28, nla_type=TCA_OPTIONS}, ",0x10,0x00,0x0a,0x00,0x05,0x00,0x00,0x00,0x0f,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x08,0x00,0x01,0x00,0x23,0x00,0x01,0x00"}]}, iov_len=72}], msg_iovlen=1, msg_controllen=0, msg_flags=0}, 0) = 72
		"app": []byte{0x10, 0x00, 0x0a, 0x00, 0x05, 0x00, 0x00, 0x00, 0x0f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x00, 0x23, 0x00, 0x01, 0x00},
	}

	for key, value := range tests {
		fmt.Println(key)
		decodeNetlink(value, 2)
	}
}
