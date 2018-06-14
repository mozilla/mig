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

	"mig.ninja/mig/modules/file"
)

// FileOptions are options that can be set to augment the specifics of a search.
type FileOptions struct {
	MaxDepth           *uint8  `json:"maxDepth"`
	MatchAll           *bool   `json:"matchAll"`
	MatchAny           *bool   `json:"matchAny"`
	MatchEntireFile    *bool   `json:"matchEntireFile"`
	MismatchingContent *string `json:"mismatchingContent"`
	Limit              *uint32 `json:"limit"`
	IncludeFileSHA256  *bool   `json:"includeFileSha256"`
	DecompressFiles    *bool   `json:"decompressFiles"`
	MaxErrors          *uint16 `json:"maxErrors"`
}

// FileSearch contains parameters used to restrict a search for files.
type FileSearch struct {
	Path                 string  `json:"path"`
	FileName             string  `json:"name"`
	Description          *string `json:"description"`
	Content              *string `json:"content"`
	MinSizeInBytes       *uint32 `json:"minSizeBytes"`
	MaxSizeInBytes       *uint32 `json:"maxSizeBytes"`
	ModifiedSinceMinutes *uint32 `json:"modifiedSinceMinutes"`
	ModifiedAfterMinutes *uint32 `json:"modifiedAfterMinutes"`
	Mode                 *string `json:"mode"`
	MD5                  *string `json:"md5"`
	SHA1                 *string `json:"sha1"`
	SHA2                 *string `json:"sha2"`
	SHA3                 *string `json:"sha3"`
}

// File contains search parameters and options to augment them.
type File struct {
	Options *FileOptions `json:"options"`
	Search  FileSearch   `json:"search"`
}

func (module *File) Name() string {
	return "file"
}

func (module *File) ToParameters() (interface{}, error) {
	description := ""
	paths := []string{module.Search.Path}
	contents := []string{}
	names := []string{module.Search.FileName}
	sizes := []string{}
	modes := []string{}
	mtimes := []string{}
	md5s := []string{}
	sha1s := []string{}
	sha2s := []string{}
	sha3s := []string{}
	maxDepth := 0.0
	maxErrors := 0.0
	matchAll := false
	macroal := false
	mismatch := []string{}
	matchLimit := 0.0
	returnSha256 := false
	decompress := false

	options, _ := json.Marshal(module.Options)
	fmt.Println(string(options))

	if module.Search.Description != nil {
		description = *module.Search.Description
	}
	if module.Search.Content != nil {
		contents = append(contents, *module.Search.Content)
	}
	if module.Search.Mode != nil {
		modes = append(modes, *module.Search.Mode)
	}
	if module.Search.MinSizeInBytes != nil {
		sizes = append(sizes, fmt.Sprintf(">%d", *module.Search.MinSizeInBytes))
	}
	if module.Search.MaxSizeInBytes != nil {
		sizes = append(sizes, fmt.Sprintf("<%d", *module.Search.MaxSizeInBytes))
	}
	if module.Search.ModifiedSinceMinutes != nil {
		mtimes = append(mtimes, fmt.Sprintf("<%dm", *module.Search.ModifiedSinceMinutes))
	}
	if module.Search.ModifiedAfterMinutes != nil {
		mtimes = append(mtimes, fmt.Sprintf(">%dm", *module.Search.ModifiedAfterMinutes))
	}
	if module.Search.MD5 != nil {
		md5s = append(md5s, *module.Search.MD5)
	}
	if module.Search.SHA1 != nil {
		sha1s = append(sha1s, *module.Search.SHA1)
	}
	if module.Search.SHA2 != nil {
		sha2s = append(sha2s, *module.Search.SHA2)
	}
	if module.Search.SHA3 != nil {
		sha3s = append(sha3s, *module.Search.SHA3)
	}
	if module.Options != nil {
		if module.Options.MaxDepth != nil {
			maxDepth = float64(*module.Options.MaxDepth)
		}
		if module.Options.MaxErrors != nil {
			maxErrors = float64(*module.Options.MaxErrors)
		}
		if module.Options.MatchAll != nil {
			matchAll = *module.Options.MatchAll
		}
		if module.Options.MatchEntireFile != nil {
			macroal = *module.Options.MatchEntireFile
		}
		if module.Options.MismatchingContent != nil {
			mismatch = append(mismatch, *module.Options.MismatchingContent)
		}
		if module.Options.Limit != nil {
			matchLimit = float64(*module.Options.Limit)
		}
		if module.Options.IncludeFileSHA256 != nil {
			returnSha256 = *module.Options.IncludeFileSHA256
		}
		if module.Options.DecompressFiles != nil {
			decompress = *module.Options.DecompressFiles
		}
	}

	search := file.Search{
		Description: description,
		Paths:       paths,
		Contents:    contents,
		Names:       names,
		Sizes:       sizes,
		Modes:       modes,
		Mtimes:      mtimes,
		MD5:         md5s,
		SHA1:        sha1s,
		SHA2:        sha2s,
		SHA3:        sha3s,
		Options: file.Options{
			MaxDepth:     maxDepth,
			MaxErrors:    maxErrors,
			MatchAll:     matchAll,
			Macroal:      macroal,
			Mismatch:     mismatch,
			MatchLimit:   matchLimit,
			ReturnSHA256: returnSha256,
			Decompress:   decompress,
		},
	}
	params := file.Parameters{
		Searches: map[string]*file.Search{
			"search": &search,
		},
	}
	return params, nil
}

func (module *File) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
