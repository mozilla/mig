// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package dispatch

import (
	"mig.ninja/mig"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
)

const (
	// StatusDispatched indicates that an action has been dispatched to the
	// MIG API.
	StatusDispatched string = "dispatched"

	// StatusNone indicates that an action's status has not changed or
	// that it cannot be determined.
	StatusNone string = ""
)

// ActionDispatcher provides a service for dispatching actions to the MIG API.
type ActionDispatcher interface {
	// Dispatch sends an action to the MIG API.
	Dispatch(mig.Action, authentication.Authenticator) error
}
