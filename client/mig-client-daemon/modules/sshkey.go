// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"bytes"
	"encoding/json"

	"mig.ninja/mig/modules/sshkey"
)

// SSHKey contains the configuration parameters required to run the SSHKey module.
type SSHKey struct {
	Path     string `json:"path"`
	MaxDepth *uint8 `json:"maxDepth"`
}

func (module *SSHKey) Name() string {
	return "sshkey"
}

func (module *SSHKey) ToParameters() (interface{}, error) {
	maxDepth := 0

	if module.MaxDepth != nil {
		maxDepth = int(*module.MaxDepth)
	}

	params := sshkey.Parameters{
		Paths:    []string{module.Path},
		MaxDepth: maxDepth,
	}

	return params, nil
}

func (module *SSHKey) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
