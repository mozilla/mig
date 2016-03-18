// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe_test

import (
	"fmt"
	"github.com/mozilla/scribe"
	"testing"
)

func TestPackageQuery(t *testing.T) {
	scribe.Bootstrap()
	scribe.TestHooks(true)
	pinfo := scribe.QueryPackages()
	for _, x := range pinfo {
		fmt.Println(x.Name, x.Version, x.Type)
	}
	if len(pinfo) != 7 {
		t.FailNow()
	}
}
