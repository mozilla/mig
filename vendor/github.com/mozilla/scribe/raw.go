// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"fmt"
)

type Raw struct {
	Identifiers []RawIdentifiers `json:"identifiers,omitempty"`
}

type RawIdentifiers struct {
	Identifier string `json:"identifier,omitempty"`
	Value      string `json:"value,omitempty"`
}

func (r *Raw) isChain() bool {
	return false
}

func (r *Raw) fireChains(d *Document) ([]evaluationCriteria, error) {
	return nil, nil
}

func (r *Raw) mergeCriteria(c []evaluationCriteria) {
}

func (r *Raw) validate(d *Document) error {
	if len(r.Identifiers) == 0 {
		return fmt.Errorf("at least one identifier must be present")
	}
	for _, x := range r.Identifiers {
		if len(x.Identifier) == 0 || len(x.Value) == 0 {
			return fmt.Errorf("identifier must include identifier and value")
		}
	}
	return nil
}

func (r *Raw) getCriteria() []evaluationCriteria {
	ret := make([]evaluationCriteria, 0)
	for _, x := range r.Identifiers {
		nc := evaluationCriteria{}
		nc.identifier = x.Identifier
		nc.testValue = x.Value
		ret = append(ret, nc)
	}
	return ret
}

func (r *Raw) prepare() error {
	return nil
}

func (r *Raw) expandVariables(v []Variable) {
}
