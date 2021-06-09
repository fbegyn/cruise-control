package main

import (
	"bytes"
	"testing"

	"github.com/florianl/go-tc"
	"github.com/spf13/viper"
)

func ViperLoadBytes(raw []byte) {
	// set config type to toml
	viper.SetConfigType("toml")

	// read in the testing config
	viper.ReadConfig(bytes.NewBuffer(raw))
}

func TestConfigFlFlow(t *testing.T) {
	// declare a testing config
	testConfig := []byte(`
[filter."testing"]
type = "flow"
filterID = 10
[filter."testing".specs]
    keys = 1
    mode = 2
    baseclass = 3
    rshift = 4
    addend = 5
    mask = 6
    xor = 7
    divisor = 8
    perturb = 9
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

	actionMap := map[string][]*tc.Action{}
	// do stuff
	attrs, err := parseFilterAttrs(flConfig, handleMap, actionMap)

	// check the returned attrs
	if attrs.Kind != "flow" {
		t.Fatalf("did get expected 'route' kind, got %v", attrs.Kind)
	}

	if attrs.Flow.Keys == nil {
		t.Fatalf("keys pointer was nil, expected %v", 1)
	}
	if *attrs.Flow.Keys != 1 {
		t.Fatalf("did get expected flow.Keys, got %v", attrs.Flow.Keys)
	}

	if attrs.Flow.Mode == nil {
		t.Fatalf("mode pointer was nil, expected %v", 2)
	}
	if *attrs.Flow.Mode != 2 {
		t.Fatalf("did not get expected flow.Mode, got %v", *attrs.Flow.Mode)
	}

	if attrs.Flow.BaseClass == nil {
		t.Fatalf("baseclass pointer was nil, expected %v", 3)
	}
	if *attrs.Flow.BaseClass != 3 {
		t.Fatalf("did not get expected flow.BaseClass, got %v", *attrs.Flow.BaseClass)
	}

	if attrs.Flow.RShift == nil {
		t.Fatalf("rshift pointer was nil, expected %v", 4)
	}
	if *attrs.Flow.RShift != 4 {
		t.Fatalf("did not get expected flow.RShift, got %v", *attrs.Flow.RShift)
	}

	if attrs.Flow.Addend == nil {
		t.Fatalf("addend pointer was nil, expected %v", 5)
	}
	if *attrs.Flow.Addend != 5 {
		t.Fatalf("did not get expected flow.Addend, got %v", *attrs.Flow.Addend)
	}

	if attrs.Flow.Mask == nil {
		t.Fatalf("mask pointer was nil, expected %v", 6)
	}
	if *attrs.Flow.Mask != 6 {
		t.Fatalf("did not get expected flow.Mask, got %v", *attrs.Flow.Mask)
	}

	if attrs.Flow.XOR == nil {
		t.Fatalf("xor pointer was nil, expected %v", 7)
	}
	if *attrs.Flow.XOR != 7 {
		t.Fatalf("did not get expected flow.XOR, got %v", *attrs.Flow.XOR)
	}

	if attrs.Flow.Divisor == nil {
		t.Fatalf("divisor pointer was nil, expected %v", 8)
	}
	if *attrs.Flow.Divisor != 8 {
		t.Fatalf("did not get expected flow.Divisor, got %v", *attrs.Flow.Divisor)
	}

	if attrs.Flow.PerTurb == nil {
		t.Fatalf("perturb pointer was nil, expected %v", 9)
	}
	if *attrs.Flow.PerTurb != 9 {
		t.Fatalf("did not get expected flow.PerTurb, got %v", *attrs.Flow.PerTurb)
	}
}

func TestConfigFlFlower(t *testing.T) {
	// declare a testing config
	testConfig := []byte(`
[filter."testing"]
type = "flower"
filterID = 10
[filter."testing".specs]
    ClassID = "testingClass"
	Indev = "testIf"
	Actions = "reject-things"          
	KeyEthDst = "38:68:93:8b:8d:55"
	KeyEthDstMask = "ff:ff:ff:ff:ff:ff"
	KeyEthSrc = "38:68:93:8b:8d:55"
	KeyEthSrcMask = "ff:ff:ff:ff:ff:ff"
	KeyEthType = 3 
	KeyIPProto = 4                
	KeyIPv4Src = "192.10.0.1"       
	KeyIPv4SrcMask = 24
	KeyIPv4Dst = "192.10.0.2"   
	KeyIPv4DstMask = 24      
	KeyTCPSrc = 9   
	KeyTCPDst = 10        
	KeyUDPSrc = 11        
	KeyUDPDst = 12   
	Flags = 12        
	KeyVlanID = 13       
	KeyVlanPrio = 14       
	KeyVlanEthType = 15     
	KeyEncKeyID = 16
	KeyEncIPv4Src = "192.10.0.1"     
	KeyEncIPv4SrcMask = 24   
	KeyEncIPv4Dst = "192.10.0.2"
	KeyEncIPv4DstMask = 24   
	KeyTCPSrcMask = 21 
	KeyTCPDstMask = 22        
	KeyUDPSrcMask = 23        
	KeyUDPDstMask = 24        
	KeySctpSrc = 25
	KeySctpDst = 26      
	KeyEncUDPSrcPort = 27     
	KeyEncUDPSrcPortMask = 28
	KeyEncUDPDstPort = 29
	KeyEncUDPDstPortMask = 30
	KeyFlags = 31
	KeyFlagsMask = 32        
	KeyIcmpv4Code = 33    
	KeyIcmpv4CodeMask = 34   
	KeyIcmpv4Type = 35
	KeyIcmpv4TypeMask = 36   
	KeyIcmpv6Code = 37
	KeyIcmpv6CodeMask = 38   
	KeyArpSIP = 39
	KeyArpSIPMask = 40       
	KeyArpTIP = 41   
	KeyArpTIPMask = 42       
	KeyArpOp = 43   
	KeyArpOpMask = 44        
	KeyMplsTTL = 45    
	KeyMplsBos = 46      
	KeyMplsTc = 47      
	KeyMplsLabel = 48       
	KeyTCPFlags = 49    
	KeyTCPFlagsMask = 50      
	KeyIPTOS = 51 
	KeyIPTOSMask = 52        
	KeyIPTTL = 53    
	KeyIPTTLMask = 54        
	KeyCVlanID = 55    
	KeyCVlanPrio = 56      
	KeyCVlanEthType = 57    
	KeyEncIPTOS = 58  
	KeyEncIPTOSMask = 59     
	KeyEncIPTTL = 60 
	KeyEncIPTTLMask = 61  
	InHwCount = 62   
`)
	handleMap := map[string]uint32{
		"testing":      1,
		"testingClass": 100,
	}

	actionMap := map[string][]*tc.Action{}

	// load in the raw bytes
	ViperLoadBytes(testConfig)

	// unmarshal into the filter config struct
	flConfig := FilterConfig{}
	err := viper.UnmarshalKey("filter.testing", &flConfig)
	if err != nil {
		t.Fatalf("failed to unmarshal config into filter config: %v", err)
	}

	// do stuff
	_, err = parseFilterAttrs(flConfig, handleMap, actionMap)

	// check the returned attrs
	// if attrs.Kind != "flow" {
	// 	t.Fatalf("did get expected 'route' kind, got %v", attrs.Kind)
	// }
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

	actionMap := map[string][]*tc.Action{}

	// do stuff
	attrs, err := parseFilterAttrs(flConfig, handleMap, actionMap)

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
