package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/florianl/go-tc"
)

type TcConfig struct {
	Qdiscs  map[string]tc.Object
	Classes map[string]tc.Object
	Filters map[string]tc.Object
}

// Parse the generated traffic file JSON back into a config file
func parseTrafficFile(file string) (TcConfig, error) {
	inp := TcConfig{}
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return inp, err
	}
	json.Unmarshal(dat, &inp)
	return inp, nil
}

// Generate a traffic file from the current TcConfig
func generateTrafficFile(conf TcConfig, file string) error {
	// render the config to the JSON file
	rawConf, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, rawConf, 0644)
	if err != nil {
		return err
	}
	return nil
}
