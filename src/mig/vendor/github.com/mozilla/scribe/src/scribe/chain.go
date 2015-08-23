// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"strings"
)

func hasChainVariables(arg string) bool {
	if strings.Contains(arg, "${chain_root}") {
		return true
	}
	return false
}

func validateChains(cl []string, d *Document) error {
	for _, x := range cl {
		_, err := d.getObjectInterface(x)
		if err != nil {
			return err
		}
	}
	return nil
}
