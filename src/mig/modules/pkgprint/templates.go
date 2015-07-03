// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package pkgprint

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
}

var templates = []ppTemplate{
	{"mediawiki", fingerprint{
		"DefaultSettings.php",
		false,
		"",
		".*wgVersion = (\\S+)",
		transformNull,
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

func transformNull(in string) (string, error) {
	return in, nil
}
