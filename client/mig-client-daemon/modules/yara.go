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
	"strings"

	"mig.ninja/mig/modules/yara"
)

// Yara contains the configuration parameters required to run the Yara module.
type Yara struct {
	Rules string   `json:"yaraRules"`
	Files []string `json:"filePaths"`
}

func (module *Yara) Name() string {
	return "yara"
}

func (module *Yara) ToParameters() (interface{}, error) {
	fileSearches := make([]string, len(module.Files))

	for index := 0; index < len(module.Files); index++ {
		fileSearches[index] = fmt.Sprintf("-path %s", module.Files[index])
	}

	params := yara.Parameters{
		YaraRules:  module.Rules,
		FileSearch: strings.Join(fileSearches, " "),
	}

	return params, nil
}

func (module *Yara) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
