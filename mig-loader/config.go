// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/mozilla/mig"
	"github.com/mozilla/mig/mig-agent/agentcontext"

	"gopkg.in/gcfg.v1"
)

type config struct {
	Loader struct {
		DiscoverPublicIP   bool
		DiscoverAWSMeta    bool
		Proxies            string
		Api                string
		Tags               []string
		RequiredSignatures int
	}
	Logging mig.Logging
}

// configDefault returns the default agent configuration file path for the
// platform.
func configDefault() string {
	return path.Join(agentcontext.GetConfDir(), "mig-loader.cfg")
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
	var globals = newGlobals()
	if err = globals.parseConfig(config); err != nil {
		panic(err)
	}
	return
}

// globals receives parsed config settings and applies them to global vars.
// newGlobals returns a Globals struct populated with initial values from global vars.
type globals struct {
	discoverPulicIP    bool              // attempt to discover the public IP of the endpoint by querying the api
	discoverAWSMeta    bool              // attempt to discover meta-data for instances running in AWS
	apiURL             string            // location of the MIG API, used for discovering the public IP
	proxies            []string          // proxies to try
	tags               map[string]string // tags for environment
	requiredSignatures int               // required signature count on manifests

	loggingConf mig.Logging
}

func newGlobals() *globals {
	return &globals{
		discoverPulicIP:    DISCOVERPUBLICIP,
		discoverAWSMeta:    DISCOVERAWSMETA,
		loggingConf:        LOGGINGCONF,
		apiURL:             APIURL,
		proxies:            PROXIES,
		tags:               TAGS,
		requiredSignatures: REQUIREDSIGNATURES,
	}
}

// stringPair takes a string separated by colon and returns key, value pair
func stringPair(pair string) (key, value string) {
	parts := strings.Split(pair, ":")
	if len(parts) == 0 {
		return "", ""
	}

	key = strings.Trim(parts[0], " ")

	if len(parts) == 1 {
		return key, ""
	}

	return key, strings.Trim(parts[1], " ")
}

// parseConfig converts config settings into usable types for global vars
// and reports errors when converting settings into go types.
func (g globals) parseConfig(config config) error {
	// In addition to any tags that have already been included in the
	// loader built-in configuration (e.g., configuration.go), also add
	// any tags specified in the configuration file if present
	if len(config.Loader.Tags) > 0 {
		for _, tag := range config.Loader.Tags {
			key, val := stringPair(tag)
			if key == "" {
				continue
			}

			g.tags[key] = val
		}
	}

	g.discoverPulicIP = config.Loader.DiscoverPublicIP
	g.discoverAWSMeta = config.Loader.DiscoverAWSMeta
	if config.Loader.RequiredSignatures != 0 {
		g.requiredSignatures = config.Loader.RequiredSignatures
	}
	g.loggingConf = config.Logging
	g.apiURL = config.Loader.Api
	if config.Loader.Proxies != "" {
		g.proxies = strings.Split(config.Loader.Proxies, ",")
	}

	// set global vars
	g.apply()
	return nil
}

// apply sets global variables with config settings.
func (g globals) apply() {
	DISCOVERPUBLICIP = g.discoverPulicIP
	DISCOVERAWSMETA = g.discoverAWSMeta
	LOGGINGCONF = g.loggingConf
	APIURL = g.apiURL
	PROXIES = g.proxies
	TAGS = g.tags
	REQUIREDSIGNATURES = g.requiredSignatures
}
