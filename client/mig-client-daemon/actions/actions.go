// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

const (
	// StatusPendingDispatch indicates that an action has not been dispatched.
	StatusPendingDispatch string = "pending-dispatch"

	// StatusDispatched indicates that an action has been dispatched to the
	// MIG API.
	StatusDispatched string = "dispatched"

	// StatusNone indicates that an action's status has not changed or
	// that it cannot be determined.
	StatusNone string = ""
)

const (
	// InvalidID represents an ID that was either not found or is invalid.
	InvalidID InternalActionID = 0
)

// InternalActionID is a descriptive alias for the type of an action's ID
// as it is represented by the MIG API.
type InternalActionID uint64
