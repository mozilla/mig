// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package config

import (
	"encoding/json"
	"os"
)

// Configuration contains parsed configuration data required to run
// the MIG Client Daemon.
type Configuration struct {
	ListenPort uint16 `json:"port"`
}

// MustLoad will try to load and parse a file at a given path into a
// `Configuration` struct and return it.  Should it fail to do so,
// this `MustLoad` will panic.
func MustLoad(path string) Configuration {
	config := Configuration{}

	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	return config
}
