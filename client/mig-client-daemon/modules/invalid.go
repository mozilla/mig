// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"errors"
)

// InvalidModule serves as a placeholder `Module` to handle cases where invalid
// module configuration data is supplied to the API.
type InvalidModule struct{}

func (module *InvalidModule) Name() string {
	return "Invalid module"
}

func (module *InvalidModule) ToParameters() (interface{}, error) {
	return nil, errors.New("Invalid module.")
}

func (module *InvalidModule) InitFromMap(_ map[string]interface{}) error {
	return errors.New("Invalid module.")
}
