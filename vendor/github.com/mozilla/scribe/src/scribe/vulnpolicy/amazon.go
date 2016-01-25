// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package vulnpolicy

import (
	"fmt"
	"scribe"
)

const amazon_expression = "^(Amazon Linux AMI.*)$"

func amazonGetReleaseTest(doc *scribe.Document, vuln Vulnerability) (string, error) {
	reltestname := fmt.Sprintf("test-release-%v-%v", vuln.OS, vuln.Release)
	relobjname := "obj-release-amazonsystemrelease"
	// See if we have a release definition for this already, if not
	// add it
	for _, x := range doc.Tests {
		if x.TestID == reltestname {
			return reltestname, nil
		}
	}

	found := false
	for _, x := range doc.Objects {
		if x.Object == relobjname {
			found = true
			break
		}
	}
	if !found {
		obj := scribe.Object{}
		obj.Object = relobjname
		obj.FileContent.Path = "/etc"
		obj.FileContent.File = "^system-release$"
		obj.FileContent.Expression = amazon_expression
		doc.Objects = append(doc.Objects, obj)
	}

	test := scribe.Test{}
	test.TestID = reltestname
	test.Object = relobjname
	test.Regexp.Value = "Amazon Linux AMI release"
	doc.Tests = append(doc.Tests, test)

	return test.TestID, nil
}
