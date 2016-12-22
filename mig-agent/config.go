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
		NoPersistMods    bool
		PersistConfigDir string
		ExtraPrivacyMode bool
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
	var globals = newGlobals()
	if err = globals.parseConfig(config); err != nil {
		panic(err)
	}
	return
}

// globals receives parsed config settings and applies them to global vars.
// newGlobals returns a Globals struct populated with initial values from global vars.
type globals struct {

	// restart the agent on failures, don't let it die
	isImmortal bool

	// request installing of a service to start the agent at boot
	mustInstallService bool

	// attempt to discover the public IP of the endpoint by querying the api
	discoverPulicIP bool

	// attempt to discover meta-data for instances running in AWS
	discoverAWSMeta bool

	// in check-in mode, the agent connects to the relay, runs all pending commands
	// and exits. this mode is used to run the agent as a cron job, not a daemon.
	checkin bool

	// if enabled, the agent will inform modules to mask returned meta-data as much
	// as possible. modules which consider this will tell you they found something,
	// but not much else.
	extraPrivacyMode bool

	// spawn persistent modules; if enabled in the built-in config this can be
	// disabled at run-time using a config option or command line flag
	spawnPersistent bool

	// directory to look in for persistent module configuration files
	persistConfigDir string

	// how often the agent will refresh its environment. if 0 agent
	// will only update environment at initialization.
	refreshEnv time.Duration

	loggingConf mig.Logging

	// location of the rabbitmq server
	// if a direct connection fails, the agent will look for the environment
	// variables HTTP_PROXY and HTTPS_PROXY, and retry the connection using
	// HTTP CONNECT proxy tunneling
	amqBroker string

	// location of the MIG API, used for discovering the public IP
	apiURL string

	// if the connection still fails after looking for a HTTP_PROXY, try to use the
	// proxies listed below
	proxies []string
	// If you don't want proxies in the built-in configuration, use the following
	// instead.
	// var PROXIES = []string{}

	// local socket used to retrieve stat information from a running agent
	socket string

	// frequency at which the agent sends heartbeat messages
	heartBeatFreq time.Duration

	// timeout after which a module run is killed
	moduleTimeout time.Duration

	// Not supported by config
	// Control modules permissions by PGP keys
	// AGENTACL [...]string

	// Not supported by config
	// PGP public keys that are authorized to sign actions
	// this is an array of strings, put each public key block
	// into its own array entry, as shown below
	// PUBLICPGPKEYS [...]string

	// CA cert that signs the rabbitmq server certificate, for verification
	// of the chain of trust. If rabbitmq uses a self-signed cert, add this
	// cert below
	caCert []byte

	// All clients share a single X509 certificate, for TLS auth on the
	// rabbitmq server. Add the public client cert below.
	agentCert []byte

	// Add the private client key below.
	agentKey []byte
}

func newGlobals() *globals {
	return &globals{
		isImmortal:         ISIMMORTAL,
		mustInstallService: MUSTINSTALLSERVICE,
		discoverPulicIP:    DISCOVERPUBLICIP,
		discoverAWSMeta:    DISCOVERAWSMETA,
		checkin:            CHECKIN,
		extraPrivacyMode:   EXTRAPRIVACYMODE,
		spawnPersistent:    SPAWNPERSISTENT,
		persistConfigDir:   MODULECONFIGDIR,
		refreshEnv:         REFRESHENV,
		loggingConf:        LOGGINGCONF,
		amqBroker:          AMQPBROKER,
		apiURL:             APIURL,
		proxies:            PROXIES,
		socket:             SOCKET,
		heartBeatFreq:      HEARTBEATFREQ,
		moduleTimeout:      MODULETIMEOUT,
		caCert:             CACERT,
		agentCert:          AGENTCERT,
		agentKey:           AGENTKEY,
	}
}

// parseConfig converts config settings into usable types for global vars
// and reports errors when converting settings into go types.
func (g globals) parseConfig(config config) error {
	var err error

	g.isImmortal = config.Agent.IsImmortal
	g.mustInstallService = config.Agent.InstallService
	g.discoverPulicIP = config.Agent.DiscoverPublicIP
	g.discoverAWSMeta = config.Agent.DiscoverAWSMeta
	g.checkin = config.Agent.CheckIn
	g.extraPrivacyMode = config.Agent.ExtraPrivacyMode
	if config.Agent.NoPersistMods {
		g.spawnPersistent = false
	}
	if config.Agent.PersistConfigDir != "" {
		g.persistConfigDir = config.Agent.PersistConfigDir
	}
	if config.Agent.RefreshEnv != "" {
		g.refreshEnv, err = time.ParseDuration(config.Agent.RefreshEnv)
		if err != nil {
			return fmt.Errorf("config.Agent.RefreshEnv %v", err)
		}
	}
	g.loggingConf = config.Logging
	g.amqBroker = config.Agent.Relay
	g.apiURL = config.Agent.Api
	if config.Agent.Proxies != "" {
		g.proxies = strings.Split(config.Agent.Proxies, ",")
	}
	g.socket = config.Agent.Socket
	g.heartBeatFreq, err = time.ParseDuration(config.Agent.HeartbeatFreq)
	if err != nil {
		return fmt.Errorf("config.Agent.HeartbeatFreq %v", err)
	}
	g.moduleTimeout, err = time.ParseDuration(config.Agent.ModuleTimeout)
	if err != nil {
		return fmt.Errorf("config.Agent.ModuleTimeout %v", err)
	}
	if config.Certs.Ca != "" {
		cacert, err := ioutil.ReadFile(config.Certs.Ca)
		if err != nil {
			return fmt.Errorf("config.Certs.Ca %v", err)
		}
		if len(cacert) > 0 {
			g.caCert = cacert
		}
	}
	if config.Certs.Cert != "" {
		agentcert, err := ioutil.ReadFile(config.Certs.Cert)
		if err != nil {
			return fmt.Errorf("config.Certs.Cert %v", err)
		}
		if len(agentcert) > 0 {
			g.agentCert = agentcert
		}
	}
	if config.Certs.Key != "" {
		agentkey, err := ioutil.ReadFile(config.Certs.Key)
		if err != nil {
			return fmt.Errorf("config.Certs.Key %v", err)
		}
		if len(agentkey) > 0 {
			g.agentKey = agentkey
		}
	}

	// set global vars
	g.apply()
	return nil
}

// apply sets global variables with config settings.
func (g globals) apply() {
	ISIMMORTAL = g.isImmortal
	MUSTINSTALLSERVICE = g.mustInstallService
	DISCOVERPUBLICIP = g.discoverPulicIP
	DISCOVERAWSMETA = g.discoverAWSMeta
	CHECKIN = g.checkin
	EXTRAPRIVACYMODE = g.extraPrivacyMode
	SPAWNPERSISTENT = g.spawnPersistent
	MODULECONFIGDIR = g.persistConfigDir
	REFRESHENV = g.refreshEnv
	LOGGINGCONF = g.loggingConf
	AMQPBROKER = g.amqBroker
	APIURL = g.apiURL
	PROXIES = g.proxies
	SOCKET = g.socket
	HEARTBEATFREQ = g.heartBeatFreq
	MODULETIMEOUT = g.moduleTimeout
	CACERT = g.caCert
	AGENTCERT = g.agentCert
	AGENTKEY = g.agentKey
}
