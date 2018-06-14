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
	Description *string `json:"description"`
	Name        *string `json:"name"`
	Library     *string `json:"library"`
	Bytes       *string `json:"bytes"`
	Content     *string `json:"content"`
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
	var description string
	var names, libraries, bytesList, contents []string

	if module.Search.Description != nil {
		description = *module.Search.Description
	}
	if module.Search.Name != nil {
		names = append(names, *module.Search.Name)
	}
	if module.Search.Library != nil {
		libraries = append(libraries, *module.Search.Library)
	}
	if module.Search.Bytes != nil {
		bytesList = append(bytesList, *module.Search.Bytes)
	}
	if module.Search.Content != nil {
		contents = append(contents, *module.Search.Content)
	}
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
		Description: description,
		Names:       names,
		Libraries:   libraries,
		Bytes:       bytesList,
		Contents:    contents,
		Options:     options,
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
