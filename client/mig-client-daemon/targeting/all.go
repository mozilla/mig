// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

import (
	"bytes"
	"encoding/json"
)

// All is a targeting `Query` used to target all online agents.
type All struct {
	All bool `json:"all"`
}

func (query *All) ToSQLWhereClause() (string, error) {
	return "status = 'online'", nil
}

func (query *All) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(query)
}
