// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	_ "mig/modules/agentdestroy"
	_ "mig/modules/file"
	_ "mig/modules/netstat"
	_ "mig/modules/timedrift"
	//_ "mig/modules/upgrade"
	_ "mig/modules/memory"
	_ "mig/modules/ping"
)
