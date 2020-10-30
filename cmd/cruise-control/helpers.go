package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
)

// StrHandle is a simple helper function that desctruct a human readable handle
func StrHandle(handle string) (uint32, error) {

	// if handle is root, return the root
	if handle == "root" {
		return tc.HandleRoot, nil
	}

	var handleMaj, handleMin int64
	var err error
	handleParts := strings.Split(handle, ":")
	handleMaj, err = strconv.ParseInt(handleParts[0], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the major part of the handle: %s\n", err)
	}
	handleMin, err = strconv.ParseInt(handleParts[1], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the minor part of the handle: %s\n", err)
	}
	return core.BuildHandle(uint32(handleMaj), uint32(handleMin)), nil
}
