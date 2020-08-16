package main

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
)

// StrHandle is a simple helper function that desctruct a human readable handle
func StrHandle(handle string) uint32 {

	// if handle is root, return the root
	if handle == "root" {
		return tc.HandleRoot
	}

	// Go on the road to match every possible way of writing a handle
	full, err := regexp.Compile("([0-9a-fA-F]+:[0-9a-fA-F]+)")
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to compile full regex")
	}
	maj, err := regexp.Compile("([0-9a-fA-F]+:)")
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to compile maj regex")
	}
	min, err := regexp.Compile("(:[0-9a-fA-F]+)")
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to compile min regex")
	}

	var handleMaj, handleMin int64
	t := strings.Split(handle, ":")
	switch true {
	case full.MatchString(handle):
		handleMaj, err = strconv.ParseInt(t[0], 16, 32)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse major part of handle")
			return 0
		}
		handleMin, err = strconv.ParseInt(t[1], 16, 32)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse minor part of handle")
			return 0
		}
	case maj.MatchString(handle):
		handleMaj, err = strconv.ParseInt(t[0], 16, 32)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse major part of handle")
			return 0
		}
	case min.MatchString(handle):
		handleMin, err = strconv.ParseInt(t[1], 16, 32)
		if err != nil {
			logger.Log("level", "ERROR", "msg", "failed to parse minor part of handle")
			return 0
		}
	}
	return core.BuildHandle(uint32(handleMaj), uint32(handleMin))
}
