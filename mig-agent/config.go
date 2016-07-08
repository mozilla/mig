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
	"strings"
	"time"

	"mig.ninja/mig"

	"gopkg.in/gcfg.v1"
)

type config struct {
	Agent struct {
		IsImmortal       bool
		InstallService   bool
		DiscoverPublicIP bool
		DiscoverAWSMeta  bool
		CheckIn          bool
		Proxies          string
		Relay            string
		Socket           string
		HeartbeatFreq    string
		ModuleTimeout    string
		Api              string
		RefreshEnv       string
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
	if err = gcfg.ReadFileInto(&config, path); err != nil {
		panic(err)
	}
	hbf, err := time.ParseDuration(config.Agent.HeartbeatFreq)
	if err != nil {
		panic(err)
	}
	timeout, err := time.ParseDuration(config.Agent.ModuleTimeout)
	if err != nil {
		panic(err)
	}
	agentkey, err := ioutil.ReadFile(config.Certs.Key)
	if err != nil {
		panic(err)
	}
	cacert, err := ioutil.ReadFile(config.Certs.Ca)
	if err != nil {
		panic(err)
	}
	agentcert, err := ioutil.ReadFile(config.Certs.Cert)
	if err != nil {
		panic(err)
	}
	var refreshenv time.Duration
	if config.Agent.RefreshEnv != "" {
		refreshenv, err = time.ParseDuration(config.Agent.RefreshEnv)
		if err != nil {
			panic(err)
		}
	}

	ISIMMORTAL = config.Agent.IsImmortal
	MUSTINSTALLSERVICE = config.Agent.InstallService
	DISCOVERPUBLICIP = config.Agent.DiscoverPublicIP
	DISCOVERAWSMETA = config.Agent.DiscoverAWSMeta
	if config.Agent.Proxies != "" {
		PROXIES = strings.Split(config.Agent.Proxies, ",")
	}
	CHECKIN = config.Agent.CheckIn
	LOGGINGCONF = config.Logging
	AMQPBROKER = config.Agent.Relay
	APIURL = config.Agent.Api
	HEARTBEATFREQ = hbf
	MODULETIMEOUT = timeout
	CACERT = cacert
	AGENTCERT = agentcert
	AGENTKEY = agentkey
	REFRESHENV = refreshenv
	return
}
