// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"mig.ninja/mig/modules/pkg"
)

// Pkg contains the configuration parameters required to run the Pkg module.
type Pkg struct {
	PackageName string  `json:"packageName"`
	Version     *string `json:"packageVersion"`
}

func (module Pkg) Name() string {
	return "pkg"
}

func (module Pkg) ToParameters() (interface{}, error) {
	version := ""

	if module.Version != nil {
		version = *module.Version
	}

	params := pkg.Parameters{
		PkgMatch: pkg.PkgMatch{
			Matches: []string{
				module.PackageName,
			},
		},
		VerMatch: version,
	}

	return params, nil
}
