// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"bytes"
	"encoding/json"
	"fmt"

	"mig.ninja/mig/modules/timedrift"
)

// TimeDrift contains the configuration parameters required to run the timedrift module.
type TimeDrift struct {
	Drift uint `json:"drift"`
}

func (module *TimeDrift) Name() string {
	return "timedrift"
}

func (module *TimeDrift) ToParameters() (interface{}, error) {
	params := timedrift.Parameters{
		Drift: fmt.Sprintf("%ds", module.Drift),
	}
	return params, nil
}

func (module *TimeDrift) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
