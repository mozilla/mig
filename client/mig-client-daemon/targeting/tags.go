// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ByTag is a targeting `Query` used to target agents based on data in an
// agent's tags.
type ByTag struct {
	TagName  string `json:"tagName"`
	TagValue string `json:"value"`
}

func (query *ByTag) ToSQLWhereClause() (string, error) {
	return fmt.Sprintf("tags->>'%s' = '%s'", query.TagName, query.TagValue), nil
}

func (query *ByTag) InitFromMap(jsonMap map[string]interface{}) error {
	encoded, err := json.Marshal(jsonMap)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(query)
}
