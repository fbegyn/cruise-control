package main

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
)

func ViperLoadBytes(raw []byte) {
	// set config type to toml
	viper.SetConfigType("toml")

	// read in the testing config
	viper.ReadConfig(bytes.NewBuffer(raw))
}

func TestConfigFlRoute(t *testing.T) {
	// declare a testing config
	testConfig := []byte(`
[filter."testing"]
type = "route"
filterID = 10
[filter."testing".specs]
    from = 2
    to = 3
    iif = 4
    classID = "testClass"
`)

	handleMap := map[string]uint32{
		"testing":   1,
		"testClass": 100,
	}

	// load in the raw bytes
	ViperLoadBytes(testConfig)

	// unmarshal into the filter config struct
	flConfig := FilterConfig{}
	err := viper.UnmarshalKey("filter.testing", &flConfig)
	if err != nil {
		t.Fatalf("failed to unmarshal config into filter config: %v", err)
	}

	// do stuff
	attrs, err := parseFilterAttrs(flConfig, handleMap)

	// check the returned attrs
	if attrs.Kind != "route" {
		t.Fatalf("did get expected 'route' kind, got %v", attrs.Kind)
	}

	if attrs.Route4.ClassID == nil {
		t.Fatalf("classID pointer was nil, expected %v", 100)
	}
	if *attrs.Route4.ClassID != 100 {
		t.Fatalf("did get expected classID, got %v", attrs.Route4.ClassID)
	}

	if attrs.Route4.From == nil {
		t.Fatalf("from pointer was nil, expected %v", 2)
	}
	if *attrs.Route4.From != 2 {
		t.Fatalf("did not get expected route.from, got %v", *attrs.Route4.From)
	}

	if attrs.Route4.To == nil {
		t.Fatalf("to pointer was nil, expected %v", 3)
	}
	if *attrs.Route4.To != 3 {
		t.Fatalf("did not get expected route.to, got %v", *attrs.Route4.To)
	}

	if attrs.Route4.IIf == nil {
		t.Fatalf("IIf pointer was nil, expected %v", 4)
	}
	if *attrs.Route4.IIf != 4 {
		t.Fatalf("did not get expected route.IIf, got %v", *attrs.Route4.IIf)
	}
}
