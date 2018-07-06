// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package search

import (
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
	"mig.ninja/mig/modules"
)

// ResultAggregator is implemented by types capable of searching for results
// produced by agents that have run a particular action.
// This action must be identified by the identifier that the MIG API uses,
// which can be retrieved by doing a lookup into a `Catalog`.
type ResultAggregator interface {
	Search(actions.InternalActionID, authentication.Authenticator) ([]modules.Result, error)
}
