// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Dustin J. Mitchell <dustin@mozilla.com>

package netstat /* import "mig.ninja/mig/modules/netstat" */

import (
	"mig.ninja/mig/testutil"
	"testing"
)

func TestRegistration(t *testing.T) {
	testutil.CheckModuleRegistration(t, "netstat")
}
