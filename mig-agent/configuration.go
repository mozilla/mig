// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package main

import (
	"github.com/mozilla/mig"
	"time"
)

// This is the agent built-in configuration. The values in this file are defaults,
// and are typically overridden using the agent configuration file. Note that however,
// these options can be modified directly before the agent is built if you wish to
// build the agent and run it without any form of external configuration.

// TAGS are useful to differentiate agents. You can add whatever values
// you want in this map, and they will be sent by the agent in each heartbeat.
var TAGS = map[string]string{}

// ISIMMORTAL controls whether or not the agent will attempt to restart itself on failure.
var ISIMMORTAL = true

// MUSTINSTALLSERVICE if true will cause the agent to install itself as a service on the
// host when it is executed.
var MUSTINSTALLSERVICE = true

// DISCOVERPUBLICIP if set to true will cause the agent to attempt to discover it's
// public IP address (e.g., if it is behind NAT) and include this in heartbeat messages.
var DISCOVERPUBLICIP = true

// DISCOVERAWSMETA if true will cause the agent to attempt to locate the AWS metadata
// service and include instance details in it's environment.
var DISCOVERAWSMETA = true

// CHECKIN if true sets the agent in check-in mode. In check-in mode, the agent will
// start up, attempt to locate any outstanding actions/commands, execute them and
// exit once all actions are responded to.
var CHECKIN = false

// EXTRAPRIVACYMODE if true informs modules that certain result data should be masked
// in the response. The implementation of privacy masking is left to the modules.
var EXTRAPRIVACYMODE = false

// SPAWNPERSISTENT if true causes the agent to start up and manage any persistent modules
// it has been built with.
var SPAWNPERSISTENT = true

// REFRESHENV controls how often the agent will refresh it's environment. If zero
// the agent will only do this once on startup.
var REFRESHENV = time.Minute * 5

// LOGGINGCONF controls the agents logging output. By default, the agent just logs
// to stdout.
var LOGGINGCONF = mig.Logging{
	Mode:  "stdout",
	Level: "info",
}

// AMQPBROKER controls the location of the RabbitMQ relay the agent will connect
// to.
var AMQPBROKER = "amqp://guest:guest@localhost:5672/"

// APIURL controls the location of the API the agent will use for public IP discovery.
var APIURL = "http://localhost:1664/api/v1/"

// PROXIES can be used to configure proxies the agent should use. Note that proxies
// can also be configured using the standard environment variables (e.g., HTTP_PROXY).
var PROXIES = []string{}

// SOCKET is the local socket the agent will listen on for control messages and status
// requests.
var SOCKET = "127.0.0.1:51664"

// HEARTBEATFREQ is the frequency at which the agent sends heartbeat messages.
var HEARTBEATFREQ = 300 * time.Second

// MODULETIMEOUT specifies the maximum time a module run should execute, after which
// the module will be killed. Note this does not apply to persistent modules which
// always execute.
var MODULETIMEOUT = 300 * time.Second

// ONLYVERIFYPUBKEY if true will cause the agent to ignore ACLs (e.g., weight comparisons
// for verification) and the agent will execute the module if a signature matches any
// key in the agents keyring.
var ONLYVERIFYPUBKEY = false

// STATSMAXACTIONS controls the number of actions the agent will store and display
// over it's status socket if queried. This can be used to view history of actions an
// agent has received.
var STATSMAXACTIONS = 15

// AGENTACL is a JSON document that describes the ACL used when executing actions from
// investigators. See the Permission type in the mig package for information on the format
// of this document.
var AGENTACL = `{
                  "default": {
                    "minimumweight": 1,
                    "investigators": {}
                  }
                }`

// PUBLICPGPKEYS is a slice of keys used to make up the agent keyring. The agents
// keyring stores public key from investigators, used to verify signatures on actions
// being sent to the agent.
var PUBLICPGPKEYS = []string{}

// CACERT is a byte slice containing the CA certificate used to validate the connection
// to the RabbitMQ relay.
var CACERT = []byte("")

// AGENTCERT is the client certificate the agent will use when connecting to the RabbitMQ
// relay.
var AGENTCERT = []byte("")

// AGENTKEY is the private key associated with AGENTCERT.
var AGENTKEY = []byte("")
