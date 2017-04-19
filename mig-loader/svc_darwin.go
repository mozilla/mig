// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"fmt"
)

func serviceMode() error {
	return fmt.Errorf("service mode not implemented for this platform")
}

func serviceTriggers() error {
	return nil
}
