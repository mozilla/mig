// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"mig.ninja/mig/modules/audit"
)

// Audit contains the configuration parameters required to run the audit module.
type Audit struct{}

func (module *Audit) Name() string {
	return "audit"
}

func (module *Audit) ToParameters() (interface{}, error) {
	return audit.Parameters{}, nil
}

func (module *Audit) InitFromMap(jsonData map[string]interface{}) error {
	return nil
}
