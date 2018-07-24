// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"github.com/mozilla/mig"
)

// TAGS are useful to differentiate agents. You can add whatever values
// you want in this map, and they will be sent by the agent in each heartbeat.
var TAGS = map[string]string{}

// DISCOVERPUBLICIP if set to true will cause the loader to attempt to discover it's
// public IP address (e.g., if it is behind NAT) and include this with it's environment.
var DISCOVERPUBLICIP = true

// DISCOVERAWSMETA if true will cause the agent to attempt to locate the AWS metadata
// service and include instance details in it's environment.
var DISCOVERAWSMETA = true

// LOADERKEY is the key used to authenticate the loader with the API when it requests
// manifest information. This value is read from /etc/mig-loader.key.
var LOADERKEY = ""

// LOGGINGCONF controls the loader logging output. By default, the loader just logs
// to stdout.
var LOGGINGCONF = mig.Logging{
	Mode:  "stdout",
	Level: "info",
}

// APIURL controls the location of the API the loader will use for manifest requests.
var APIURL = "http://localhost:1664/api/v1/"

// PROXIES can be used to configure proxies the loader should use. Note that proxies
// can also be configured using the standard environment variables (e.g., HTTP_PROXY).
var PROXIES = []string{}

// REQUIREDSIGNATURES indicates the number of valid signatures a manifest must have for
// it to be considered for installation.
var REQUIREDSIGNATURES = 1

// MANIFESTPGPKEYS is a slice of PGP public keys the loader will use to verify signatures
// that are applied to a manifest.
var MANIFESTPGPKEYS = []string{}
