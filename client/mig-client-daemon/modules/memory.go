// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"bytes"
	"encoding/json"

	memory "mig.ninja/mig/modules/memory"
)

type MemoryOptions struct {
	Offset      *uint64 `json:"offset"`
	MaxLength   *uint64 `json:"maxLength"`
	LogFailures *bool   `json:"logFailures"`
	MatchAll    *bool   `json:"matchAll"`
}

type MemorySearch struct {
	Names     []string `json:"names"`
	Libraries []string `json:"libraries"`
	Bytes     []string `json:"bytes"`
	Contents  []string `json:"contents"`
}

type Memory struct {
	Options *MemoryOptions `json:"options"`
	Search  MemorySearch   `json:"search"`
}

func (module *Memory) Name() string {
	return "memory"
}

func (module *Memory) ToParameters() (interface{}, error) {
	var offset, maxLength uint64
	var logFailures, matchAll bool

	if module.Options != nil {
		if module.Options.Offset != nil {
			offset = *module.Options.Offset
		}
		if module.Options.MaxLength != nil {
			maxLength = *module.Options.MaxLength
		}
		if module.Options.LogFailures != nil {
			logFailures = *module.Options.LogFailures
		}
		if module.Options.MatchAll != nil {
			matchAll = *module.Options.MatchAll
		}
	}

	options := memory.Options{
		Offset:      float64(offset),
		MaxLength:   float64(maxLength),
		LogFailures: logFailures,
		MatchAll:    matchAll,
	}
	search := memory.Search{
		Names:     module.Search.Names,
		Libraries: module.Search.Libraries,
		Bytes:     module.Search.Bytes,
		Contents:  module.Search.Contents,
		Options:   options,
	}
	params := memory.Parameters{
		Searches: map[string]memory.Search{
			"search": search,
		},
	}

	return params, nil
}

func (module *Memory) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
