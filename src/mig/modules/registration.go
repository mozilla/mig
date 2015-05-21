// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Dustin J. Mitchell <dustin@mozilla.com>
//              Julien Vehent jvehent@mozilla.com [:ulfr]

package modules

// Stores details about the registration of a module
type Registration struct {
	Runner func() interface{}
}

// Available stores a list of activated module with their registration
var Available = make(map[string]Registration)

// Register adds a module to the list of available modules
func Register(name string, runner func() interface{}) {
	if _, exist := Available[name]; exist {
		panic("Register: a module named " + name + " has already been registered.\nAre you trying to import the same module twice?")
	}
	newmodule := &Registration{}
	newmodule.Runner = runner
	Available[name] = *newmodule
}
