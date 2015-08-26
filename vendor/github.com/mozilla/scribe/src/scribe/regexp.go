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

type regex struct {
	Value string `json:"value"`
}

func (r *regex) evaluate(c evaluationCriteria) (ret evaluationResult, err error) {
	var re *regexp.Regexp
	debugPrint("evaluate(): regexp %v \"%v\", \"%v\"\n", c.identifier, c.testValue, r.Value)
	re, err = regexp.Compile(r.Value)
	if err != nil {
		return
	}
	ret.criteria = c
	if re.MatchString(c.testValue) {
		ret.result = true
	}
	return
}
