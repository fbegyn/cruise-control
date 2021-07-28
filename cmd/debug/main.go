package main

import (
	"fmt"
	"strings"

	"github.com/mdlayher/netlink"
)

const nlafNested = (1 << 15)

func main() {
	tests := map[string][]byte{
		// sendmsg(3, {msg_name={sa_family=AF_NETLINK, nl_pid=0, nl_groups=00000000}, msg_namelen=12, msg_iov=[{iov_base={{nlmsg_len=92, nlmsg_type=RTM_NEWTFILTER, nlmsg_flags=NLM_F_REQUEST|NLM_F_ACK|NLM_F_EXCL|NLM_F_CREATE, nlmsg_seq=1625691781, nlmsg_pid=0}, {tcm_family=AF_UNSPEC, tcm_ifindex=if_nametoindex("test-01"), tcm_handle=0, tcm_parent=65536, tcm_info=768}, [{{nla_len=8, nla_type=TCA_KIND}, "u32"}, {{nla_len=48, nla_type=TCA_OPTIONS}, ",0x10,0x00,0x0a,0x00,0x01,0x00,0x00,0x00,0x0f,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x08,0x00,0x01,0x00,0x11,0x00,0x01,0x00,0x14,0x00,0x05,0x00,0x01,0x00,0x00,0x00"...}]}, iov_len=92}], msg_iovlen=1, msg_controllen=0, msg_flags=0}, 0) = 92
		"iproute": []byte{0x10, 0x00, 0x0a, 0x00, 0x01, 0x00, 0x00, 0x00, 0x0f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x00, 0x11, 0x00, 0x01, 0x00, 0x14, 0x00, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00},
		// sendmsg(3, {msg_name={sa_family=AF_NETLINK, nl_pid=0, nl_groups=00000000}, msg_namelen=12, msg_iov=[{iov_base={{nlmsg_len=72, nlmsg_type=RTM_NEWTFILTER, nlmsg_flags=NLM_F_REQUEST|NLM_F_ACK|NLM_F_CREATE, nlmsg_seq=563752620, nlmsg_pid=28466}, {tcm_family=AF_UNSPEC, tcm_ifindex=if_nametoindex("test-01"), tcm_handle=0, tcm_parent=65536, tcm_info=0}, [{{nla_len=8, nla_type=TCA_KIND}, "u32"}, {{nla_len=28, nla_type=TCA_OPTIONS}, ",0x10,0x00,0x0a,0x00,0x05,0x00,0x00,0x00,0x0f,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x08,0x00,0x01,0x00,0x23,0x00,0x01,0x00"}]}, iov_len=72}], msg_iovlen=1, msg_controllen=0, msg_flags=0}, 0) = 72
		"app": []byte{0x10, 0x00, 0x0a, 0x00, 0x05, 0x00, 0x00, 0x00, 0x0f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x00, 0x23, 0x00, 0x01, 0x00},
	}

	for key, value := range tests {
		fmt.Println(key)
		decodeNetlink(value, 2)
	}
}

func decodeNetlink(data []byte, lvl int) error {
	ad, err := netlink.NewAttributeDecoder(data)
	if err != nil {
		return err
	}
	for ad.Next() {
		space := strings.Repeat(" ", lvl)
		if ad.Type()&nlafNested == nlafNested {
			fmt.Printf("%s %4d | %d\n", space, ad.Type(), ad.Type()^nlafNested)
			decodeNetlink(ad.Bytes(), lvl+3)
			continue
		}
		fmt.Printf("%s %4d\t%#v\n", space, ad.Type(), ad.Bytes())
	}
	return nil
}
