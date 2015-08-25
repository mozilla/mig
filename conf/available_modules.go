// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	_ "mig.ninja/mig/modules/agentdestroy"
	_ "mig.ninja/mig/modules/file"
	_ "mig.ninja/mig/modules/memory"
	_ "mig.ninja/mig/modules/netstat"
	_ "mig.ninja/mig/modules/ping"
	_ "mig.ninja/mig/modules/pkg"
	_ "mig.ninja/mig/modules/scribe"
	_ "mig.ninja/mig/modules/timedrift"
	//_ "mig/modules/upgrade"
	//_ "mig/modules/example"
)
