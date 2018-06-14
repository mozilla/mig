// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"bytes"
	"encoding/json"

	"mig.ninja/mig/modules/agentdestroy"
)

// AgentDestroy contains the configuration parameters required to run the agentdestroy module.
type AgentDestroy struct {
	PID     uint   `json:"pid"`
	Version string `json:"version"`
}

func (module *AgentDestroy) Name() string {
	return "agentdestroy"
}

func (module *AgentDestroy) ToParameters() (interface{}, error) {
	params := agentdestroy.Parameters{
		PID:     int(module.PID),
		Version: module.Version,
	}
	return params, nil
}

func (module *AgentDestroy) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
