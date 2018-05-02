// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

import (
	"errors"
)

// InvalidQuery serves as a placeholder `Query` to handle cases where invalid
// targeting data is supplied to the API.
type InvalidQuery struct{}

func (target InvalidQuery) ToSQLWhereClause() (string, error) {
	return "", errors.New("Invalid target.")
}

func (target *InvalidQuery) InitFromMap(jsonData map[string]interface{}) error {
	return errors.New("Invalid target.")
}
