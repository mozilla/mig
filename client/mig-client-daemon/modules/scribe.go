// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"bytes"
	"encoding/json"
	"os"

	scribelib "github.com/mozilla/scribe"
	"mig.ninja/mig/modules/scribe"
)

// Scribe contains the configuration parameters required to run the scribe module.
type Scribe struct {
	Path                string `json:"path"`
	OnlyTrueDocTests    *bool  `json:"onlyTrueDocTests"`
	HumanReadableOutput *bool  `json:"humanReadableOutput"`
	JSONOutput          *bool  `json:"jsonOutput"`
}

func (module *Scribe) Name() string {
	return "scribe"
}

func (module *Scribe) ToParameters() (interface{}, error) {
	var onlyTrue, humanReadable, jsonOutput bool

	scribeFile, openErr := os.Open(module.Path)
	if openErr != nil {
		return scribe.Parameters{}, openErr
	}
	defer scribeFile.Close()
	scribeDoc := scribelib.Document{}
	decoder := json.NewDecoder(scribeFile)
	decodeErr := decoder.Decode(&scribeDoc)
	if decodeErr != nil {
		return scribe.Parameters{}, decodeErr
	}

	if module.OnlyTrueDocTests != nil {
		onlyTrue = *module.OnlyTrueDocTests
	}
	if module.HumanReadableOutput != nil {
		humanReadable = *module.HumanReadableOutput
	}
	if module.JSONOutput != nil {
		jsonOutput = *module.JSONOutput
	}

	params := scribe.Parameters{
		ScribeDoc:   scribeDoc,
		OnlyTrue:    onlyTrue,
		HumanOutput: humanReadable,
		JSONOutput:  jsonOutput,
	}
	return params, nil
}

func (module *Scribe) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
