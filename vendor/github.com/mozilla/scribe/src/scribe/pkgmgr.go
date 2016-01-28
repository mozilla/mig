// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"os/exec"
	"regexp"
	"strings"
)

var pkgmgrInitialized bool
var pkgmgrCache []pkgmgrInfo

type pkgmgrResult struct {
	results []pkgmgrInfo
}

type pkgmgrInfo struct {
	name    string
	version string
	pkgtype string
	arch    string
}

// Package information from the system as returned by QueryPackages().
type PackageInfo struct {
	Name    string `json:"name"`    // Package name.
	Version string `json:"version"` // Package version.
	Type    string `json:"type"`    // Package type.
	Arch    string `json:"arch"`    // Package architecture
}

// Query packages on the system, returning a slice of all identified packages
// in PackageInfo form.
func QueryPackages() []PackageInfo {
	ret := make([]PackageInfo, 0)
	for _, x := range getAllPackages().results {
		np := PackageInfo{}
		np.Name = x.name
		np.Version = x.version
		np.Type = x.pkgtype
		np.Arch = x.arch
		ret = append(ret, np)
	}
	return ret
}

func getPackage(name string, collectexp string) (ret pkgmgrResult) {
	ret.results = make([]pkgmgrInfo, 0)
	if !pkgmgrInitialized {
		pkgmgrInit()
	}
	debugPrint("getPackage(): looking for \"%v\"\n", name)
	for _, x := range pkgmgrCache {
		if collectexp == "" {
			if x.name != name {
				continue
			}
		} else {
			mtch, err := regexp.MatchString(collectexp, x.name)
			if err != nil || !mtch {
				continue
			}
		}
		debugPrint("getPackage(): found %v, %v, %v\n", x.name, x.version, x.pkgtype)
		ret.results = append(ret.results, x)
	}
	debugPrint("getPackage(): returning %v entries\n", len(ret.results))
	return
}

func getAllPackages() pkgmgrResult {
	ret := pkgmgrResult{}
	ret.results = make([]pkgmgrInfo, 0)
	if !pkgmgrInitialized {
		pkgmgrInit()
	}
	for _, x := range pkgmgrCache {
		ret.results = append(ret.results, x)
	}
	return ret
}

func pkgmgrInit() {
	debugPrint("pkgmgrInit(): initializing package manager...\n")
	pkgmgrCache = make([]pkgmgrInfo, 0)
	if sRuntime.testHooks {
		pkgmgrCache = append(pkgmgrCache, testGetPackages()...)
	} else {
		pkgmgrCache = append(pkgmgrCache, rpmGetPackages()...)
		pkgmgrCache = append(pkgmgrCache, dpkgGetPackages()...)
	}
	pkgmgrInitialized = true
	debugPrint("pkgmgrInit(): initialized with %v packages\n", len(pkgmgrCache))
}

func rpmGetPackages() []pkgmgrInfo {
	ret := make([]pkgmgrInfo, 0)

	c := exec.Command("rpm", "-qa", "--queryformat", "%{NAME} %{EVR} %{ARCH}\\n")
	buf, err := c.Output()
	if err != nil {
		return ret
	}

	slist := strings.Split(string(buf), "\n")
	for _, x := range slist {
		s := strings.Fields(x)

		if len(s) < 3 {
			continue
		}
		newpkg := pkgmgrInfo{}
		newpkg.name = s[0]
		newpkg.version = s[1]
		newpkg.arch = s[2]
		newpkg.pkgtype = "rpm"
		ret = append(ret, newpkg)
	}
	return ret
}

func dpkgGetPackages() []pkgmgrInfo {
	ret := make([]pkgmgrInfo, 0)

	c := exec.Command("dpkg", "-l")
	buf, err := c.Output()
	if err != nil {
		return nil
	}

	slist := strings.Split(string(buf), "\n")
	for _, x := range slist {
		s := strings.Fields(x)

		if len(s) < 4 {
			continue
		}
		// Only process packages that have been fully installed.
		if s[0] != "ii" {
			continue
		}
		newpkg := pkgmgrInfo{}
		newpkg.name = s[1]
		newpkg.version = s[2]
		newpkg.arch = s[3]
		newpkg.pkgtype = "dpkg"
		ret = append(ret, newpkg)
	}
	return ret
}

// Functions and data related to package tests

var testPkgTable = []struct {
	name string
	ver  string
}{
	{"openssl", "1.0.1e"},
	{"bash", "4.3-11"},
	{"upstart", "1.13.2"},
	{"grub-common", "2.02-beta2"},
	{"libbind", "1:9.9.5.dfsg-4.3"},
	{"kernel", "2.6.32-504.12.2.el6.x86_64"},
	{"kernel", "2.6.32-573.8.1.el6.x86_64"},
}

func testGetPackages() []pkgmgrInfo {
	ret := make([]pkgmgrInfo, 0)
	for _, x := range testPkgTable {
		newpkg := pkgmgrInfo{}
		newpkg.name = x.name
		newpkg.version = x.ver
		newpkg.pkgtype = "test"
		ret = append(ret, newpkg)
	}
	return ret
}
