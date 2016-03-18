// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

// This function does criteria concatenation based on identifier. Where
// criteria is found with a matching identifier, the results are
// concatenated together. Since the slice is ordered based on expression
// groups there should not be any ordering issues.
func criteriaConcat(in []evaluationCriteria, concat string) []evaluationCriteria {
	ret := make([]evaluationCriteria, 0)
	if len(in) == 0 {
		return ret
	}
	debugPrint("criteriaConcat(): applying concat with \"%v\" on identifier\n", concat)
	retmap := make(map[string]evaluationCriteria, 0)
	for _, x := range in {
		nr := evaluationCriteria{identifier: x.identifier}
		retmap[x.identifier] = nr
	}
	for _, x := range in {
		retent := retmap[x.identifier]
		if len(retmap[x.identifier].testValue) == 0 {
			retent.testValue = x.testValue
		} else {
			retent.testValue = retent.testValue + concat + x.testValue
		}
		retmap[x.identifier] = retent
	}
	for _, x := range retmap {
		debugPrint("criteriaConcat(): result \"%v\", \"%v\"\n", x.identifier, x.testValue)
		ret = append(ret, x)
	}
	return ret
}
