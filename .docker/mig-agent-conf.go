// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import(
	"mig"
	"time"

    _ "mig/modules/filechecker"
    _ "mig/modules/netstat"
    _ "mig/modules/upgrade"
    _ "mig/modules/agentdestroy"
    _ "mig/modules/example"
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

// attempt to discover the public IP of the endpoint by querying a STUN server
var DISCOVERPUBLICIP = false

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
var AMQPBROKER string = "amqp://agent:SomeRandomAgentPassword@localhost:5672/"

// if the connection still fails after looking for a HTTP_PROXY, try to use the
// proxies listed below
var PROXIES = [...]string{`proxy.example.net:3128`, `proxy2.example.net:8080`}

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
            "Test test": {
                "fingerprint": "3DFA3D6A289CD378D960F943C4010ABF3C91AF53",
                "weight": 3
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
Version: GnuPG v1; test@example.com

mI0EVCF3XQEEANquhquH8GoTGrYLATYvP1X8lgnfN2q3j58k2/WNS8XM1YanIEhu
NYJPy+oS+nR9FsXF+kiMlJEiv8ez+yOYXA9NBNDrKa8n6P5JouTVbgtSsDzeTK5n
6hr5h07gk3VTKyr/q1zHTSnaHopKQQIKJzr5kHi/iPqiK4Hb+rFVYcJHABEBAAG0
M0pvaG4gRG9lIChJbnNlY3VyZSBrZXkgZm9yIHRlc3QpIDx0ZXN0QGV4YW1wbGUu
Y29tPoi4BBMBAgAiBQJUIXddAhsDBgsJCAcDAgYVCAIJCgsEFgIDAQIeAQIXgAAK
CRDEAQq/PJGvUyOVA/kBLaNsJ3y8HAasaUJx6429p1keEXGBWyTv0D22Yre/SkZR
VCmsJ2ZXL3WhI3EQzjKxlwW4NHGOdDYH80DOfGdrqG2mt/tfMmDWxEtAi3of2eCk
ZvaBfSWVu3Z/UNi9eHIK1+GHIb4zfQgki82+rDNqAD0AgDf5UgoO5Rr+KPd+07iN
BFQhd10BBACo3RVONn1a0/F4GHAEyHr2wmMygP/caY2WewqevXvh+8aOGjNgdJUY
uxPu/7oH3A6fYJSnqiWCSXiZW+t0m+9lqVbYwBLjmq9gTnl1b6zwbWeWwS9E0tvR
FAFHJG38l7S4RGgEKC7V4RI5ZcVd2Iph1pmvTrBDN98HPawbUiZpFQARAQABiJ8E
GAECAAkFAlQhd10CGwwACgkQxAEKvzyRr1MxwAP/Rfulvxn+JE+ka6Ji2UDROdf5
OBoltR0ptYq6XNLRpwRmMHYg5ZVQJ6/QMfirMmt2gCOu5w9R8dfn5UTwal6z3JqZ
jNfSNHwFuSk/EfkzhPdlXSs+Zc/vaPWeXsM1tnlFCmbSo1X+ujZbK6DemAap76VQ
75sK74uzHxugIPShvNU=
=AyG6
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
