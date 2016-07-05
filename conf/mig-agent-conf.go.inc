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

// attempt to discover meta-data for instances running in AWS
var DISCOVERAWSMETA = true

// in check-in mode, the agent connects to the relay, runs all pending commands
// and exits. this mode is used to run the agent as a cron job, not a daemon.
var CHECKIN = false

// how often the agent will refresh its environment. if 0 agent
// will only update environment at initialization.
var REFRESHENV time.Duration = 0

var LOGGINGCONF = mig.Logging{
	Mode:	"stdout",	// stdout | file | syslog
	Level:	"debug",	// debug | info | ...
	//File:	"/tmp/migagt.log",
	//MaxFileSize: 0,
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
var PROXIES = []string{"proxy.example.net:3128", "proxy2.example.net:8080"}
// If you don't want proxies in the built-in configuration, use the following
// instead.
// var PROXIES = []string{}

// local socket used to retrieve stat information from a running agent
var SOCKET = "127.0.0.1:51664"

// frequency at which the agent sends heartbeat messages
var HEARTBEATFREQ time.Duration = 300 * time.Second

// timeout after which a module run is killed
var MODULETIMEOUT time.Duration = 300 * time.Second

// Control modules permissions by PGP keys
var AGENTACL = [...]string{
`{
    "default": {
        "minimumweight": 2,
        "investigators": {
            "Bob Kelso": {
                "fingerprint": "E60892BB....",
                "weight": 2
            },
            "Morpheus": {
                "fingerprint": "AD595634....",
                "weight": 3
            }
        }
    }
}`,
`{
    "agentdestroy": {
        "minimumweight": 1,
        "investigators": {
            "MIG Scheduler": {
                "fingerprint": "E60892BB...",
                "weight": 1
            }
        }
    }
}`,
}


// PGP public keys that are authorized to sign actions
// this is an array of strings, put each public key block
// into its own array entry, as shown below
var PUBLICPGPKEYS = [...]string{
`
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1; Name: User for MIG test (Another test user for Mozilla Investigator) <usertest+mig@example.org>

mI0EUvJc0gEEAJuW77RlSYpAa777tI1foSVB6Vxp7XVE6fe7lmc6PokvMHjKZCB9
.........
lMVXz7c/B8T79KIH0EDAG8o6AbvZQdTMSZp+Ap562smLkV+xsPo1O1Zd/hDJKYuY
936oKqajBV4Jh8vXGb3r
=SWyb
-----END PGP PUBLIC KEY BLOCK-----
`,
`
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1; Name: Test User (This is a test user for Mozilla Investigator) <testuser+mig@example.net>

mI0EUvJcngEEAKH4MbzljzAha4MzUy4wnNHqNX65hlsWD3wPMAPL4R0F8h9VuyLw
.........
vld2mOto/1HZ7I3re0ItO/M+kpn1VgcsWFTmunohlmAZUKh9LK6gGZ4nXEqe3Lbx
QnD9SDA9/d80
=phhK
-----END PGP PUBLIC KEY BLOCK-----
`}


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
