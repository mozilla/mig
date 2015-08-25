// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

type exactmatch struct {
	Value string `json:"value"`
}

func (e *exactmatch) evaluate(c evaluationCriteria) (ret evaluationResult, err error) {
	debugPrint("evaluate(): exactmatch %v \"%v\", \"%v\"\n", c.identifier, c.testValue, e.Value)
	ret.criteria = c
	if c.testValue == e.Value {
		ret.result = true
	}
	return
}
