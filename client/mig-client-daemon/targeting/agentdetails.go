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
	"fmt"
	"strings"
)

// ByAgentDetails is a targeting `Query` used to target agents based on data
// specific to a given agent.
// In order for this `Query` to be considered valid, at least one of the fields
// in the struct must be provided.
type ByAgentDetails struct {
	ID            *uint   `json:"id"`
	Name          *string `json:"name"`
	QueueLocation *string `json:"queueLocation"`
	Version       *string `json:"version"`
	Pid           *uint   `json:"pid"`
	Status        *string `json:"status"`
}

func (query *ByAgentDetails) ToSQLWhereClause() (string, error) {
	allNil := query.ID == nil &&
		query.Name == nil &&
		query.QueueLocation == nil &&
		query.Version == nil &&
		query.Pid == nil &&
		query.Status == nil

	if allNil {
		return "", errors.New("Cannot target agents by agent details without providing at least one field to match against.")
	}

	queryStrings := []string{}
	if query.ID != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("id = %d", *query.ID))
	}
	if query.Name != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("name LIKE '%s'", *query.Name))
	}
	if query.QueueLocation != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("queueloc LIKE '%s'", *query.QueueLocation))
	}
	if query.Version != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("version LIKE '%s'", *query.Version))
	}
	if query.Pid != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("pid = %d", *query.Pid))
	}
	if query.Status != nil {
		queryStrings = append(queryStrings, fmt.Sprintf("status = '%s'", *query.Status))
	}

	queryString := strings.Join(queryStrings, " AND ")
	return queryString, nil
}

func (query *ByAgentDetails) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	return decoder.Decode(query)
}
