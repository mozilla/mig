// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"mig.ninja/mig"
)

type Context struct {
	AgentIdentifier mig.Agent

	Channels struct {
		Log chan mig.Log
	}
	Logging mig.Logging
}
