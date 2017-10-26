// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

// Version can be set at compile time to indicate the version of MIG
// components. You'd typically want to set this during install using flags
// such as -ldflags "-X mig.ninja/mig.Version=20170913-0.06824ce0.dev" when
// calling the go build tools.
var Version = ""

func init() {
	// If the default value of Version is not being specified using the build
	// tools, just set a generic version identifier.
	if Version == "" {
		Version = "0.unversioned"
	}
}
