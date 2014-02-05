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

import(
	"mig"
	"time"
)

var LOGGINGCONF = mig.Logging{
	Mode:	"stdout",	// stdout | file | syslog
	Level:	"debug",	// debug | info | ...
	//File:	"/tmp/migagt.log",
	//Host:	"syslog_hostname",
	//Port:	514,
	//Protocol: "udp",
}

// location of the rabbitmq server
var AMQPBROKER string = "amqp://guest:guest@localhost:5672/"

// frequency at which the agent sends keepalive messages
var HEARTBEATFREQ string = "300s"

// timeout after which a module run is killed
var MODULETIMEOUT time.Duration = 300 * time.Second

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
var AMQPCACERT string = `-----BEGIN CERTIFICATE-----
MIIHyTCCBbGgAwIBAgIBATANBgkqhkiG9w0BAQUFADB9MQswCQYDVQQGEwJJTDEW
........
NOsF/5oirpt9P/FlUQqmMGqz9IgcgA38corog14=
-----END CERTIFICATE-----
`

// All clients share a single X509 certificate, for TLS auth on the
// rabbitmq server. Add the public client cert below.
var AGENTCERT string = `-----BEGIN CERTIFICATE-----
MIIGYjCCBUqgAwIBAgIDDD5PMA0GCSqGSIb3DQEBBQUAMIGMMQswCQYDVQQGEwJJ
........
04lr0kZCZTYpIQ5KFFe/s+3n0A3RDu4qzhrxOf3BMHyAITB+/Nh4IlRCZu2ygv2X
ej2w/mPv
-----END CERTIFICATE-----
`

// Add the private client key below.
var AGENTKEY string = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAvJQqCjE4I63S3kR9KV0EG9e/lX/bZxa/2QVvZGi9/Suj65nD
........
RMSEpg+wuIVnKUi6KThiMKyXfZaTX7BDuR/ezE/JHs1TN5Hkw43TCQ==
-----END RSA PRIVATE KEY-----
`
