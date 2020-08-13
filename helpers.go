package main

import (
	"strconv"
	"strings"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
)

// StrHandle is a simple helper function that desctruct a human readable handle
func StrHandle(handle string) uint32 {
	t := strings.Split(handle, ":")
	if t[0] == "root" {
		return tc.HandleRoot
	} else {
		handleMaj, err := strconv.ParseInt(t[0], 10, 32)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse major part of handle")
		}
		handleMin, err := strconv.ParseInt(t[1], 10, 32)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse minor part of handle")
		}
		return core.BuildHandle(uint32(handleMaj), uint32(handleMin))
	}
}
