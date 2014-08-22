// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"io/ioutil"
	"mig"
	"time"

	"code.google.com/p/gcfg"
)

type config struct {
	Agent struct {
		IsImmortal       bool
		InstallService   bool
		DiscoverPublicIP bool
		Relay            string
		Socket           string
		HeartbeatFreq    string
		ModuleTimeout    string
	}
	Certs struct {
		Ca, Cert, Key string
	}
	Logging mig.Logging
}

// configLoad reads a local configuration file and overwrite the global conf
// variable with the parameters from the file
func configLoad(path string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("configLoad() -> %v", e)
		}
	}()
	var config config
	err = gcfg.ReadFileInto(&config, path)
	if err != nil {
		panic(err)
	}
	ISIMMORTAL = config.Agent.IsImmortal
	MUSTINSTALLSERVICE = config.Agent.InstallService
	DISCOVERPUBLICIP = config.Agent.DiscoverPublicIP
	LOGGINGCONF = config.Logging
	AMQPBROKER = config.Agent.Relay
	HEARTBEATFREQ, err = time.ParseDuration(config.Agent.HeartbeatFreq)
	if err != nil {
		panic(err)
	}
	MODULETIMEOUT, err = time.ParseDuration(config.Agent.ModuleTimeout)
	if err != nil {
		panic(err)
	}
	CACERT, err = ioutil.ReadFile(config.Certs.Ca)
	if err != nil {
		panic(err)
	}
	AGENTCERT, err = ioutil.ReadFile(config.Certs.Cert)
	if err != nil {
		panic(err)
	}
	AGENTKEY, err = ioutil.ReadFile(config.Certs.Key)
	if err != nil {
		panic(err)
	}
	return
}
