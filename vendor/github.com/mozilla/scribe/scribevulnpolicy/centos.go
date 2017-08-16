// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"fmt"
	"github.com/mozilla/scribe"
)

type centosRelease struct {
	identifier   int
	versionMatch string
	expression   string
	filematch    string
}

const centosExpression = ".*CentOS.*(release \\d+)\\..*"

var centosReleases = []centosRelease{
	{platformCentos7, "release 7", centosExpression, "^centos-release$"},
	{platformCentos6, "release 6", centosExpression, "^centos-release$"},
}

// The list of packages for this platform we will only consider the newest version for in the
// generated policy
var centosOnlyNewestPackages = []string{
	"kernel",
	"kernel-abi-whitelists",
	"kernel-headers",
	"kernel-devel",
	"kernel-debug",
	"kernel-debug-devel",
	"kernel-debuginfo",
	"kernel-debuginfo-common",
	"kernel-doc",
	"kernel-tools",
	"kernel-tools-debuginfo",
	"kernel-tools-libs",
	"perf",
	"perf-debuginfo",
	"python-perf",
	"python-perf-debuginfo",
}

// In some cases we only want to collect version information on the latest installed version
// of a package to use for tests. For example, on CentOS we may have multiple versions of
// "kernel" installed but we only want to test against the latest version so we don't get a
// bunch of false positives.
//
// This function returns true if this is the case.
func centosOnlyNewest(pkgname string) bool {
	for _, x := range centosOnlyNewestPackages {
		if x == pkgname {
			return true
		}
	}
	return false
}

// Adds a release test to scribe document doc. The release test is a dependency
// for each other vuln check, and validates if a given package is vulnerable that the
// platform is also what is expected (e.g., package X is vulnerable and operating system
// is also X.
func centosReleaseTest(platform supportedPlatform, doc *scribe.Document) (tid string, err error) {
	var (
		test    scribe.Test
		obj     scribe.Object
		release centosRelease
	)

	// Set the name and referenced object for the release test
	test.TestID = fmt.Sprintf("test-release-%v", platform.name)
	test.Object = "test-release"

	// Set our match value on the test to the release string
	found := false
	for _, x := range centosReleases {
		if x.identifier == platform.platformID {
			found = true
			release = x
			break
		}
	}
	if !found {
		err = fmt.Errorf("unable to locate release version match for %v", platform.name)
		return
	}
	test.EMatch.Value = release.versionMatch

	// Add our object, which will be the file we will match against to determine
	// if the platform is in scope
	obj.Object = test.Object
	obj.FileContent.Path = "/etc"
	obj.FileContent.File = release.filematch
	obj.FileContent.Expression = release.expression

	doc.Tests = append(doc.Tests, test)
	doc.Objects = append(doc.Objects, obj)
	tid = test.TestID
	return
}
