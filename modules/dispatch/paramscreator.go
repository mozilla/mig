// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package dispatch /* import "github.com/mozilla/mig/modules/dispatch" */

import (
	"fmt"
)

func printHelp(isCmd bool) {
	fmt.Printf(`Query parameters
----------------
This module has no parameters.
`)
}

// ParamsParser parses any parameters used in queries for this module.
func (r *run) ParamsParser(args []string) (interface{}, error) {
	return r.Parameters, r.ValidateParameters()
}
