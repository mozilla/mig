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

	"mig.ninja/mig/modules/ping"
)

// Ping contains the configuration parameters required to run the Ping module.
type Ping struct {
	Destination     string  `json:"destination"`
	Protocol        *string `json:"protocol"`
	DestinationPort *uint16 `json:"destinationPort"`
	Count           *uint   `json:"count"`
	Timeout         *uint   `json:"timeout"`
}

func (module *Ping) Name() string {
	return "ping"
}

func (module *Ping) ToParameters() (interface{}, error) {
	// Set up defaults
	dest := module.Destination
	protocol := "tcp"
	port := 0.0
	count := 0.0
	timeout := 0.0

	if module.Protocol != nil {
		protocol = *module.Protocol
	}
	if module.DestinationPort != nil {
		port = float64(*module.DestinationPort)
	}
	if module.Count != nil {
		count = float64(*module.Count)
	}
	if module.Timeout != nil {
		timeout = float64(*module.Timeout)
	}

	params := ping.Params{
		Destination:     dest,
		DestinationPort: port,
		Protocol:        protocol,
		Count:           count,
		Timeout:         timeout,
	}

	fmt.Println("Produced Params instance for ping module", params)

	return params, nil
}

func (module *Ping) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
