// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

import (
	"errors"
	"fmt"
)

// Query is implemented by types that can be converted into a string
// containing a condition for a SQL query's `WHERE` clause, used to
// target agents that we want to run an action.
type Query interface {
	ToSQLWhereClause() (string, error)
	InitFromMap(map[string]interface{}) error
}

// FromMap attempts to populate a `Query` with data from a `map` containing
// targeting parameters parsed from JSON.
func FromMap(jsonMap map[string]interface{}) (Query, error) {
	queryContainers := []Query{
		new(ByAgentDetails),
		new(ByHostDetails),
		new(ByTag),
	}

	fmt.Printf("trying to load query from data %v\n", jsonMap)
	for _, query := range queryContainers {
		fmt.Printf("Attempting to decode target into %T\n", query)
		err := query.InitFromMap(jsonMap)
		if err == nil {
			fmt.Println("Succeeded")
			return query, nil
		}
		fmt.Printf("Failed. Error = %s\n", err.Error())
	}

	return new(InvalidQuery), errors.New("Not a recognized agent target query.")
}
