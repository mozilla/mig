// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

// Pkg contains the configuration parameters required to run the Pkg module.
type Pkg struct {
	Name    string
	Version *string
}

func (module Pkg) Validate() error {
	return nil
}
