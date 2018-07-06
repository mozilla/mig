// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import "mig.ninja/mig/modules/dispatch"

// Dispatch contains the configuration parameters required to run the dispatch module.
type Dispatch struct{}

func (module *Dispatch) Name() string {
	return "dispatch"
}

func (module *Dispatch) ToParameters() (interface{}, error) {
	return dispatch.Parameters{}, nil
}

func (module *Dispatch) InitFromMap(jsonData map[string]interface{}) error {
	return nil
}
