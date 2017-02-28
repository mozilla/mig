// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package audit /* import "mig.ninja/mig/modules/audit" */

import (
	"fmt"
)

func initializeAudit(cfg config) error {
	return fmt.Errorf("audit module not supported on darwin")
}

func runAudit() error {
	return fmt.Errorf("audit module not supported on darwin")
}
