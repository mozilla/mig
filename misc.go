// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package mig /* import "mig.ninja/mig" */

// Misc support functions used in various places within MIG

import (
	mrand "math/rand"
	"time"
)

// RandAPIKeyString is used for prefix and key generation, and just
// returns a random string consisting of alphanumeric characters of
// length characters long
func RandAPIKeyString(length int) string {
	ret := make([]byte, length)
	lset := []byte("abcdefghijklmnopqrstuvwxyzABCDEFCHIJKLMNOPQRSTUVWXYZ0123456789")
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	for i := 0; i < len(ret); i++ {
		ret[i] = lset[r.Int()%len(lset)]
	}
	return string(ret[:len(ret)])
}
