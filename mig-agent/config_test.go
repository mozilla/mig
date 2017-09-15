// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Rob Murtha robmurtha@gmail.com [:robmurtha]

package main

import (
	"fmt"
	"testing"

	"gopkg.in/gcfg.v1"
)

func TestConfigLoadDefault(t *testing.T) {
	// loading the default config produces an error
	expect := `configLoad() -> config.Certs.Ca open /path/to/ca/cert: no such file or directory`
	err := configLoad("../conf/mig-agent.cfg.inc")
	if err == nil || err.Error() != expect {
		t.Error("expected", expect, "got", err)
	}
}

// TestConfigLoadNonExist verifies globals set in the agent build stay intact if we attempt to
// load a nonexistent configuration file
func TestConfigLoadNonExist(t *testing.T) {
	origAmqpBroker := AMQPBROKER
	origAgentCert := AGENTCERT
	AMQPBROKER = "amqp broker"
	AGENTCERT = []byte("agent cert")

	err := configLoad("/non/existent")
	if err == nil {
		t.Fatalf("loading nonexistent configuration should have failed")
	}
	if AMQPBROKER != "amqp broker" {
		t.Error("original AMQPBROKER value not intact")
	}
	if string(AGENTCERT) != "agent cert" {
		t.Error("original AGENTCERT value not intact")
	}
	AMQPBROKER = origAmqpBroker
	AGENTCERT = origAgentCert
}

// TestConfigLoadCerts verifies that certificates are loaded into global variables
// from the configuration file
func TestConfigLoadCerts(t *testing.T) {
	path := `../conf/mig-agent.cfg.inc`
	var config config
	err := gcfg.ReadFileInto(&config, path)
	if err != nil {
		t.Error("expected to read", path, "got", err)
		t.FailNow()
	}

	// Reset the global agent variables we will check later
	AGENTCERT = []byte("")
	AGENTKEY = []byte("")
	CACERT = []byte("")

	globals := globals{}
	globals.tags = make(map[string]string)
	config.Certs.Key = "../conf/mig-agent.cfg.inc"
	config.Certs.Ca = "../conf/mig-agent.cfg.inc"
	config.Certs.Cert = "../conf/mig-agent.cfg.inc"
	expect := `no error`
	err = globals.parseConfig(config)
	if err != nil {
		t.Error("expected", expect, "got", err)
	}
	expect = "AGENTCERT not empty"
	if len(AGENTCERT) == 0 {
		t.Error("expected", expect)
	}
	expect = "AGENTKEY not empty"
	if len(AGENTKEY) == 0 {
		t.Error("expected", expect)
	}
	expect = "CACERT not empty"
	if len(CACERT) == 0 {
		t.Error("expected", expect)
	}
}

// TestConfigLoadEmptyCerts verifies that if a configuration file contains an empty
// path for certificate related config options, we keep the defaults in the built-in
// configuration
func TestConfigLoadEmptyCerts(t *testing.T) {
	AGENTCERT = []byte("agent cert")
	AGENTKEY = []byte("agent key")
	CACERT = []byte("ca cert")
	globals := newGlobals()

	expect := "caCert not empty"
	if len(globals.caCert) == 0 {
		t.Error("expected", expect)
	}
	expect = "agentCert not empty"
	if len(globals.agentCert) == 0 {
		t.Error("expected", expect)
	}
	expect = "agentKey not empty"
	if len(globals.agentKey) == 0 {
		t.Error("expected", expect)
	}

	path := `../conf/mig-agent.cfg.inc`
	var config config
	err := gcfg.ReadFileInto(&config, path)
	if err != nil {
		t.Error("expected to read", path, "got", err)
		t.FailNow()
	}

	// Simulate empty paths for certificate values
	config.Certs.Ca = ""
	config.Certs.Cert = ""
	config.Certs.Key = ""

	expect = "no error"
	err = globals.parseConfig(config)
	if err != nil {
		t.Error("expected", expect, "got", err)
	}

	// Verify our original defaults are intact
	if string(CACERT) != "ca cert" {
		t.Error("expected original CACERT value")
	}
	if string(AGENTCERT) != "agent cert" {
		t.Error("expected original AGENTCERT value")
	}
	if string(AGENTKEY) != "agent key" {
		t.Error("expected original AGENTKEY value")
	}
}

// TestConfigLoadCertErrors verifies configuration loading fails if an invalid certificate
// path is present in the configuration file.
func TestConfigLoadCertErrors(t *testing.T) {
	path := `../conf/mig-agent.cfg.inc`
	var config config
	expect := `no error`
	err := gcfg.ReadFileInto(&config, path)
	if err != nil {
		t.Error("expected", expect, "got", err)
		t.FailNow()
	}

	// start with empty globals for testing vs populated via NewGlobals
	globals := &globals{}
	globals.tags = make(map[string]string)

	expect = `config.Certs.Ca open /path/to/ca/cert: no such file or directory`
	err = globals.parseConfig(config)
	if err.Error() != expect {
		t.Error(fmt.Sprintf("expected %v got %v", expect, err))
	}
	config.Certs.Ca = ""

	expect = `config.Certs.Cert open /path/to/client/cert: no such file or directory`
	err = globals.parseConfig(config)
	if err == nil || err.Error() != expect {
		t.Error("expected", expect, "got", err)
	}
	config.Certs.Cert = ""

	expect = `config.Certs.Key open /path/to/private/key: no such file or directory`
	err = globals.parseConfig(config)
	if err == nil || err.Error() != expect {
		t.Error("expected", expect, "got", err)
	}
	config.Certs.Key = ""

	expect = "no error"
	err = globals.parseConfig(config)
	if err != nil {
		t.Error("expected", expect, "got", err)
	}
}

// TestConfigParseDurationErrors verifies we get an error indicating an invalid
// time specification for malformed duration related configuration arguments.
func TestConfigParseDurationErrors(t *testing.T) {
	var config config

	globals := &globals{}
	globals.tags = make(map[string]string)

	path := `../conf/mig-agent.cfg.inc`
	expect := `no error`
	err := gcfg.ReadFileInto(&config, path)
	if err != nil {
		t.Error("expected", expect, "got", err)
		t.FailNow()
	}
	config.Certs.Ca = ""
	config.Certs.Cert = ""
	config.Certs.Key = ""

	config.Agent.RefreshEnv = "300"
	expect = `config.Agent.RefreshEnv time: missing unit in duration 300`
	err = globals.parseConfig(config)
	if err == nil || err.Error() != expect {
		t.Error("expected", expect, "got", err)
	}
	config.Agent.RefreshEnv = "300s"

	config.Agent.HeartbeatFreq = "300"
	expect = `config.Agent.HeartbeatFreq time: missing unit in duration 300`
	err = globals.parseConfig(config)
	if err == nil || err.Error() != expect {
		t.Error("expected", expect, "got", err)
	}
	config.Agent.HeartbeatFreq = "300s"

	config.Agent.ModuleTimeout = "300"
	expect = `config.Agent.ModuleTimeout time: missing unit in duration 300`
	err = globals.parseConfig(config)
	if err == nil || err.Error() != expect {
		t.Error("expected", expect, "got", err)
	}
}
