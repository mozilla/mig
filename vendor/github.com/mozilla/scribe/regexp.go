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

// Regex is used to specify regular expression matching criteria within a test.
type Regex struct {
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

func (r *Regex) evaluate(c evaluationCriteria) (ret evaluationResult, err error) {
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
