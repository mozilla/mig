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

func TestConfigLoadCerts(t *testing.T) {
	// test that configured cert files are loaded
	path := `../conf/mig-agent.cfg.inc`
	var config config
	err := gcfg.ReadFileInto(&config, path)
	if err != nil {
		t.Error("expected to read", path, "got", err)
		t.FailNow()
	}

	globals := globals{}
	config.Certs.Key = "../conf/mig-agent.cfg.inc"
	config.Certs.Ca = "../conf/mig-agent.cfg.inc"
	config.Certs.Cert = "../conf/mig-agent.cfg.inc"
	expect := `no error`
	err = globals.parseConfig(config)
	if err != nil {
		t.Error("expected", expect, "got", err)
	}
	expect = `agentCert not empty`
	if len(AGENTCERT) == 0 {
		t.Error("expected", expect)
	}
	expect = `agentKey not empty`
	if len(AGENTKEY) == 0 {
		t.Error("expected", expect)
	}
	expect = `caCert not empty`
	if len(CACERT) == 0 {
		t.Error("expected", expect)
	}
}

func TestConfigLoadEmptyCerts(t *testing.T) {
	// test that empty certs are ok and the defaults are used
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

	config.Certs.Ca = ""
	config.Certs.Cert = ""
	config.Certs.Key = ""

	expect = "no error"
	err = globals.parseConfig(config)
	if err != nil {
		t.Error("expected", expect, "got", err)
	}

	//verify defaults are intact
	expect = "caCert not empty"
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
}

func TestConfigLoadCertErrors(t *testing.T) {
	// test that an informative error is returned for invalid cert paths
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

func TestConfigParseDurationErrors(t *testing.T) {
	// test that an informative error is returned for invalid durations
	var config config
	var globals globals

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
