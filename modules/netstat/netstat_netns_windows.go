// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package netstat /* import "mig.ninja/mig/modules/netstat" */

func namespacesSupported() bool {
	return false
}

func setNamespace(fd int) (err error) {
	return
}

func cacheNamespaces() (ret []int, err error) {
	return
}
