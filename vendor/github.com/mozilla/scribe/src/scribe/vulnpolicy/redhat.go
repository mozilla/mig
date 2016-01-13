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

type RedHatRelease struct {
	Name    string // Release identifier
	Version string // Release version
}

var RedHatReleases = []RedHatRelease{
	{"rhel7", "release 7"},
	{"rhel6", "release 6"},
	{"rhel5", "release 5"},
	{"centos7", "release 7"},
	{"centos6", "release 6"},
	{"centos5", "release 5"},
}

const rhl_expression = ".*Red Hat.*(release \\d+)\\..*"
const centos_expression = ".*CentOS.*(release \\d+)\\..*"

func redhatGetReleaseTest(doc *scribe.Document, vuln Vulnerability) (string, error) {
	reltestname := fmt.Sprintf("test-release-%v-%v", vuln.OS, vuln.Release)
	relobjname := "obj-release-redhatrelease"
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
		obj.FileContent.File = "^redhat-release$"
		if vuln.OS == "redhat" {
			obj.FileContent.Expression = rhl_expression
		} else {
			obj.FileContent.Expression = centos_expression
		}
		doc.Objects = append(doc.Objects, obj)
	}

	mvalue := ""
	for _, x := range RedHatReleases {
		if x.Name == vuln.Release {
			mvalue = x.Version
			break
		}
	}
	if mvalue == "" {
		return "", fmt.Errorf("unknown redhat/centos release %v", vuln.Release)
	}
	test := scribe.Test{}
	test.TestID = reltestname
	test.Object = relobjname
	test.EMatch.Value = mvalue
	doc.Tests = append(doc.Tests, test)

	return test.TestID, nil
}
