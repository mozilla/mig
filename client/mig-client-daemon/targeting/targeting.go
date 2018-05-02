// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Query is implemented by types that can be converted into a string
// containing a condition for a SQL query's `WHERE` clause, used to
// target agents that we want to run an action.
type Query interface {
	ToSQLWhereClause() (string, error)
}

// FromMap attempts to populate a `Query` with data from a `map` containing
// targeting parameters parsed from JSON.
func FromMap(jsonMap map[string]interface{}) (Query, error) {
	queryContainers := []Query{
		ByAgentDetails{},
		ByHostDetails{},
		ByTag{},
	}

	encoded, encodeErr := json.Marshal(jsonMap)
	if encodeErr != nil {
		return InvalidQuery{}, encodeErr
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))

	for _, query := range queryContainers {
		err := decoder.Decode(&query)
		if err == nil {
			return query, nil
		}
	}

	return InvalidQuery{}, errors.New("Not a recognized agent target query.")
}
