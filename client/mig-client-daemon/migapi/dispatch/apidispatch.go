// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package migapi

import (
	"mig.ninja/mig"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
)

// APIDispatcher is a `Dispatcher` that will send actions to the MIG API.
type APIDispatcher struct {
}

// NewAPIDispatcher constructs a new `APIDispatcher`.
func NewAPIDispatcher() APIDispatcher {
	return APIDispatcher{}
}

func (dispatch APIDispatcher) Dispatch(
	action mig.Action,
	auth authentication.Authenticator,
) error {
	return nil
}
