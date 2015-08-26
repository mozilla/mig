// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Dustin J. Mitchell <dustin@mozilla.com>
//              Julien Vehent jvehent@mozilla.com [:ulfr]

package modules /* import "mig.ninja/mig/modules" */

// A mig module implements this interface
type Moduler interface {
	NewRun() Runner
}

// The set of registered modules
var Available = make(map[string]Moduler)

// Register a new module as available
func Register(name string, mod Moduler) {
	if _, exist := Available[name]; exist {
		panic("Register: a module named " + name + " has already been registered.\nAre you trying to import the same module twice?")
	}
	Available[name] = mod
}
