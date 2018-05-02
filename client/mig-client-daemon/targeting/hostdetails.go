// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

import (
	"errors"
	"fmt"
	"strings"
)

// ByHostDetails is a targeting `Query` used to target agents based on
// information about the host they are running on.
// In order for this `Query` to be considered valid, at least one of the fields
// in the struct must be provided.
type ByHostDetails struct {
	Ident    *string `json:"ident"`
	OS       *string `json:"os"`
	Arch     *string `json:"arch"`
	PublicIP *string `json:"publicIP"`
}

func (query ByHostDetails) ToSQLWhereClause() (string, error) {
	allNil := query.Ident == nil &&
		query.OS == nil &&
		query.Arch == nil &&
		query.PublicIP == nil

	if allNil {
		return "", errors.New("Cannot target agents by host details without providing at least one field to match against.")
	}

	queryStrings := []string{}
	if query.Ident != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("environment->>'ident' = '%s'", *query.Ident))
	}
	if query.OS != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("environment->>'os' = '%s'", *query.OS))
	}
	if query.Arch != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("environment->>'arch' = '%s'", *query.Arch))
	}
	if query.PublicIP != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("environment->>'publicip' = '%s'", *query.PublicIP))
	}

	queryString := strings.Join(queryStrings, " AND ")
	return queryString, nil
}
