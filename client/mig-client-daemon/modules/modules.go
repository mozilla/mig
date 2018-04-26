// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

// Module is implemented by types that contain parameters for a module
// supported by the MIG agent.  The `ToParameters` method is expected to
// validate the module configuration data and return a data type that
// can be set as the `Parameters` field in an `Action`'s
// [Operation.Parameters](https://github.com/mozilla/mig/blob/master/action.go#L84)
// field.
type Module interface {
	Name() string
	ToParameters() (interface{}, error)
}
