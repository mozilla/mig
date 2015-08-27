// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Dustin J. Mitchell <dustin@mozilla.com>

package testutil /* import "mig.ninja/mig/testutil" */

import (
	"mig.ninja/mig/modules"
	"testing"
)

func CheckModuleRegistration(t *testing.T, module_name string) {
	mod, ok := modules.Available[module_name]
	if !ok {
		t.Fatalf("module %s not registered", module_name)
	}

	// test getting a run instance (just don't fail!)
	mod.NewRun()
}
