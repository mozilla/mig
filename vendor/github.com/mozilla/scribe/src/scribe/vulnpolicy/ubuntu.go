// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package vulnpolicy

import (
	"fmt"
	"regexp"
	"scribe"
)

type UbuntuRelease struct {
	Name    string // Release identifier
	Version string // Release version
}

var UbuntuReleases = []UbuntuRelease{
	{"wily", "15.10"},
	{"vivid", "15.04"},
	{"utopic", "14.10"},
	{"trusty", "14.04"},
	{"precise", "12.04"},
	{"lucid", "10.04"},
}

const lsb_expression = "DISTRIB_RELEASE=(\\d{1,2}\\.\\d{1,2})"

type ubuntuPackageCollect struct {
	PkgName    string // Package name
	CollectExp string // Collection expression
}

var ubuntuCollectPkg = []ubuntuPackageCollect{
	{"linux-image-generic", "^linux-image-\\d.*-generic$"},
	{"linux-image-extra-generic", "^linux-image-extra-\\d.*-generic$"},
	{"linux-headers", "^linux-headers-[\\d-.]+$"},
	{"linux-headers-generic", "^linux-headers-[\\d-.]+-generic$"},
}

func ubuntuGetReleasePackage(vuln Vulnerability) (string, string) {
	for _, x := range ubuntuCollectPkg {
		mtch, err := regexp.MatchString(x.CollectExp, vuln.Package)
		if err != nil {
			panic(err)
		}
		if mtch {
			return x.PkgName, x.CollectExp
		}
	}
	return vuln.Package, ""
}

func ubuntuGetReleaseTest(doc *scribe.Document, vuln Vulnerability) (string, error) {
	reltestname := fmt.Sprintf("test-release-%v-%v", vuln.OS, vuln.Release)
	relobjname := "obj-release-lsbrelease"
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
		obj.FileContent.File = "^lsb-release$"
		obj.FileContent.Expression = lsb_expression
		doc.Objects = append(doc.Objects, obj)
	}

	mvalue := ""
	for _, x := range UbuntuReleases {
		if x.Name == vuln.Release {
			mvalue = x.Version
			break
		}
	}
	if mvalue == "" {
		return "", fmt.Errorf("unknown ubuntu release %v", vuln.Release)
	}
	test := scribe.Test{}
	test.TestID = reltestname
	test.Object = relobjname
	test.EMatch.Value = mvalue
	doc.Tests = append(doc.Tests, test)

	return test.TestID, nil
}
