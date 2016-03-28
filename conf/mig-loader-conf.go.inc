// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

// some tags that are useful to differentiate agents. You can add whatever
// you want in this struct, it will be sent by the agent in each heartbeat
var TAGS = struct {
	Operator string `json:"operator"`
}{
	"MyFavoriteAdminTeam",
}

// attempt to discover the public IP of the endpoint by querying the api
var DISCOVERPUBLICIP = true

// attempt to discover meta-data for instances running in AWS
var DISCOVERAWSMETA = true

// loader key
var LOADERKEY = "secret"

// location of the MIG API, used for discovering the public IP
var APIURL string = "http://localhost:1664/api/v1/"

// if the connection still fails after looking for a HTTP_PROXY, try to use the
// proxies listed below
var PROXIES = [...]string{`proxy.example.net:3128`, `proxy2.example.net:8080`}

// Number of valid manifest signatures required to be acceptable
var REQUIREDSIGNATURES = 1

// PGP keys we accept manifest signatures from
var MANIFESTPGPKEYS = [...]string{
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
