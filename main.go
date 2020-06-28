package main

import (
	"fmt"
	"os"

	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
)

type Config struct {
	DownloadSpeed uint32
	UploadSpeed   uint32

	Qdiscs  map[string]Qdisc
	Classes map[string]Class
}

type Qdisc struct {
	Type  string
	Specs map[string]uint32
}

type Class struct {
	Type  string
	Specs map[string]uint32
}

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./")

	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

	if err := viper.ReadInConfig(); err != nil {
		logger.Log("level", "ERROR", "msg", "failed to read config file", "error", err)
	}
	fmt.Printf("%s\n", viper.AllSettings())

	t := Config{}
	viper.Unmarshal(&t)
	fmt.Println(t)

	for handle, qd := range t.Qdiscs {
		logger.Log("handle", handle, "type", qd.Type)
		switch qd.Type {
		case "fq_codel":
			logger.Log("msg", "creating fq_codel qdisc")
			fqcodel := tc.FqCodel{}
			fqcodel.CEThreshold = qd.Specs["cethreshold"]
			fqcodel.DropBatchSize = qd.Specs["dropbatchsize"]
			fqcodel.ECN = qd.Specs["ecn"]
			fqcodel.Flows = qd.Specs["flows"]
			fqcodel.Interval = qd.Specs["interval"]
			fqcodel.Limit = qd.Specs["limit"]
			fqcodel.MemoryLimit = qd.Specs["memorylimit"]
			fqcodel.Quantum = qd.Specs["quantum"]
			fqcodel.Target = qd.Specs["target"]
			fmt.Println(fqcodel)
		case "hfsc":
			hfsc := tc.HfscQOpt{}
			hfsc.DefCls = uint16(qd.Specs["defcls"])
			fmt.Println(hfsc)
		}
	}
}
