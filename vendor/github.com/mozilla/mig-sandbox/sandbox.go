// +build !linux
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributors:
// Alexandru Tudorica <tudalex@gmail.com>
// vladimirdiaconescu <vladimirdiaconescu@users.noreply.github.com>
// Teodora Baluta <teobaluta@gmail.com>

package sandbox

import "log"

var ActTrap = 1
var ActAllow = 2

type FilterAction string

type FilterOperation struct {
	FilterOn   []string
	Action     int
	Conditions []int
}

type SandboxProfile struct {
	DefaultPolicy int
	Filters       []FilterOperation
}

func Jail(sandboxProfile SandboxProfile) {
	log.Printf("No seccomp sandbox is available for this platform.")
}
