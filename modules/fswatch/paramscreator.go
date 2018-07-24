// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package fswatch /* import "github.com/mozilla/mig/modules/fswatch" */

import (
	"fmt"
)

func printHelp(isCmd bool) {
	fmt.Printf(`Query parameters
----------------
This module has no parameters.
`)
}

func (r *run) ParamsParser(args []string) (interface{}, error) {
	return r.Parameters, r.ValidateParameters()
}
