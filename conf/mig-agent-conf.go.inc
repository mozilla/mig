// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import(
	"mig.ninja/mig"
	"time"
)

// some tags that are useful to differentiate agents. You can add whatever
// you want in this struct, it will be sent by the agent in each heartbeat
var TAGS = struct {
	Operator string `json:"operator"`
}{
	"MyFavoriteAdminTeam",
}

// restart the agent on failures, don't let it die
var ISIMMORTAL bool = true

// request installing of a service to start the agent at boot
var MUSTINSTALLSERVICE bool = true

// attempt to discover the public IP of the endpoint by querying the api
var DISCOVERPUBLICIP = true

// in check-in mode, the agent connects to the relay, runs all pending commands
// and exits. this mode is used to run the agent as a cron job, not a daemon.
var CHECKIN = false

var LOGGINGCONF = mig.Logging{
	Mode:	"stdout",	// stdout | file | syslog
	Level:	"debug",	// debug | info | ...
	//File:	"/tmp/migagt.log",
	//Host:	"syslog_hostname",
	//Port:	514,
	//Protocol: "udp",
}

// location of the rabbitmq server
// if a direct connection fails, the agent will look for the environment
// variables HTTP_PROXY and HTTPS_PROXY, and retry the connection using
// HTTP CONNECT proxy tunneling
var AMQPBROKER string = "amqp://guest:guest@localhost:5672/"

// location of the MIG API, used for discovering the public IP
var APIURL string = "http://localhost:1664/api/v1/"

// if the connection still fails after looking for a HTTP_PROXY, try to use the
// proxies listed below
var PROXIES = [...]string{`proxy.example.net:3128`, `proxy2.example.net:8080`}

// local socket used to retrieve stat information from a running agent
var SOCKET = "127.0.0.1:51664"

// frequency at which the agent sends heartbeat messages
var HEARTBEATFREQ time.Duration = 300 * time.Second

// timeout after which a module run is killed
var MODULETIMEOUT time.Duration = 300 * time.Second

// === Access Control List ===

// Master Keys are used to verify ACL Signatures
var ACLKEYS = [...]string{
`E60892BB9BD89A69F759A1A0A3D652173B763E8F`,
`FA5D79F95F7AF7097C3E83DA26A86D5E5885AC11`}

// Signatures indicates how many signatures from master keys
// are required to accept an ACL
var ACLSIGMIN int = 1

// MemberOf indicates which groups the agent is a member of.
// For each group, an ACL will be requested from the API, verified
// and loaded into the agent.
var ACL = [...]string{`default`, `mygroup1`}

// CA cert that signs the rabbitmq server certificate, for verification
// of the chain of trust. If rabbitmq uses a self-signed cert, add this
// cert below
var CACERT = []byte(`-----BEGIN CERTIFICATE-----
MIIHyTCCBbGgAwIBAgIBATANBgkqhkiG9w0BAQUFADB9MQswCQYDVQQGEwJJTDEW
........
NOsF/5oirpt9P/FlUQqmMGqz9IgcgA38corog14=
-----END CERTIFICATE-----`)

// All clients share a single X509 certificate, for TLS auth on the
// rabbitmq server. Add the public client cert below.
var AGENTCERT = []byte(`-----BEGIN CERTIFICATE-----
MIIGYjCCBUqgAwIBAgIDDD5PMA0GCSqGSIb3DQEBBQUAMIGMMQswCQYDVQQGEwJJ
........
04lr0kZCZTYpIQ5KFFe/s+3n0A3RDu4qzhrxOf3BMHyAITB+/Nh4IlRCZu2ygv2X
ej2w/mPv
-----END CERTIFICATE-----`)

// Add the private client key below.
var AGENTKEY = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAvJQqCjE4I63S3kR9KV0EG9e/lX/bZxa/2QVvZGi9/Suj65nD
........
RMSEpg+wuIVnKUi6KThiMKyXfZaTX7BDuR/ezE/JHs1TN5Hkw43TCQ==
-----END RSA PRIVATE KEY-----`)
