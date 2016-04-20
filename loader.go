// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package mig /* import "mig.ninja/mig" */

import (
	"time"
)

// Describes a loader entry stored in the database
type LoaderEntry struct {
	ID        float64   // Loader ID
	Name      string    // Loader name
	Key       string    // Loader key (only populated during creation)
	AgentName string    // Loader environment, agent name
	LastSeen  time.Time // Last time loader was used
	Enabled   bool      // Loader entry is active
}

func (le *LoaderEntry) Validate() error {
	return nil
}
