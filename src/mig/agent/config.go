/* Mozilla InvestiGator Agent

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

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
		IsImmortal     bool
		InstallService bool
		Relay          string
		Socket         string
		HeartbeatFreq  string
		ModuleTimeout  string
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
