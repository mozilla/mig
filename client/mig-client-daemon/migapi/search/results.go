// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package search

import (
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/modules"
)

// APIResultAggregator is a `ResultAggregator` that will search for results
// from the MIG API.
type APIResultAggregator struct {
	baseAddress string
}

// NewAPIResultAggregator constructs a new `APIResultAggregator`.
func NewAPIResultAggregator(baseAddr string) APIResultAggregator {
	return APIResultAggregator{
		baseAddress: baseAddr,
	}
}

// Search queries the MIG API until it reads all of the results generated as
// a result of an action being executed by agents.
func (aggregator APIResultAggregator) Search(
	actionID actions.InternalActionID,
) ([]modules.Result, error) {
	return []modules.Result{}, nil
}
