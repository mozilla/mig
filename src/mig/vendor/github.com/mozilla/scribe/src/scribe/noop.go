// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

type noop struct {
}

func (n *noop) evaluate(c evaluationCriteria) (ret evaluationResult, err error) {
	ret.criteria = c
	ret.result = true
	return
}
