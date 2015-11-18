// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"regexp"
)

type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func variableExpansion(v []Variable, in string) string {
	res := in
	for _, x := range v {
		s := "\\$\\{" + x.Key + "\\}"
		re := regexp.MustCompile(s)
		res = re.ReplaceAllLiteralString(res, x.Value)
	}
	debugPrint("variableExpansion(): %v -> %v\n", in, res)
	return res
}
