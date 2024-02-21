package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/florianl/go-tc"
	"github.com/spf13/viper"
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

	// open a go-tc socket
	rtnl, err := CreateTcSocket()
	if err != nil {
		logger.Error("failed to open rtnetlink socket", "err", err)
		return
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			logger.Error("failed to close rtnetlink socket", "err", err)
			return
		}
		logger.Debug("closed rtnetlink socket")
	}()
	logger.Debug("opened rtnetlink socket")

	mux := http.NewServeMux()
	// handle general operations
	mux.HandleFunc("GET /api/v1/tc/{interface}", ObjectListHandler)
	mux.HandleFunc("POST /api/v1/tc/{interface}", ObjectCreateHandler)
	// handle specific operations
	mux.HandleFunc("GET /api/v1/tc/{interface}/{handle}", ObjectGetHandler)
	mux.HandleFunc("UPDATE /api/v1/tc/{interface}/{handle}", ObjectUpdateHandler)
	mux.HandleFunc("PUT /api/v1/tc/{interface}/{handle}", ObjectUpdateHandler)

	logger.Info("starting API server", "addr", conf.Addr)
	if err := http.ListenAndServe(conf.Addr, mux); err != nil {
		logger.Error("cannot start API server", "err", err)
		os.Exit(5)
	}
}
