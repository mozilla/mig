// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import "mig.ninja/mig/modules/fswatch"

// FSWatch contains configuration parameters required to run the FSWatch module.
type FSWatch struct{}

func (module *FSWatch) Name() string {
	return "fswatch"
}

func (module *FSWatch) ToParameters() (interface{}, error) {
	return fswatch.Parameters{}, nil
}

func (module *FSWatch) InitFromMap(jsonData map[string]interface{}) error {
	return nil
}
