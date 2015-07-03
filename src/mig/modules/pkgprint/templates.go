// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package pkgprint

import (
	"fmt"
	"regexp"
)

type ppTemplate struct {
	name        string
	fingerprint fingerprint
}

type fingerprint struct {
	filename     string
	isRegexp     bool
	pathFilter   string
	contentMatch string
	transform    func(string) (string, error)
	forceRoot    string
}

var templates = []ppTemplate{
	{"mediawiki", fingerprint{
		"DefaultSettings.php",
		false,
		"",
		".*wgVersion = (\\S+)",
		transformNull,
		"",
	}},
	{"django", fingerprint{
		"__init__.py",
		false,
		"django",
		"^VERSION = (\\S+, \\S+, \\S+, \\S+, \\S+)",
		transformDjango,
		"",
	}},
	{"linuxkernel", fingerprint{
		"version",
		false,
		"",
		"^(Linux version.*)",
		transformNull,
		"/proc",
	}},
	{"pythonegg", fingerprint{
		"PKG-INFO",
		false,
		"egg-info",
		"^Version: (\\S+)",
		transformNull,
		"",
	}},
}

func getTemplateFingerprint(name string) *fingerprint {
	for x := range templates {
		if templates[x].name == name {
			return &templates[x].fingerprint
		}
	}
	return nil
}

func transformDjango(in string) (string, error) {
	re := regexp.MustCompile("^\\((\\S+), (\\S+), (\\S+),")
	buf := re.FindStringSubmatch(in)
	if len(buf) != 4 {
		return "", fmt.Errorf("transform django: invalid input \"%v\"", in)
	}
	ret := fmt.Sprintf("%v.%v.%v", buf[1], buf[2], buf[3])
	return ret, nil
}

func transformNull(in string) (string, error) {
	return in, nil
}
