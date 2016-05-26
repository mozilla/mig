// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package mig /* import "mig.ninja/mig" */

import (
	"errors"
	"regexp"
	"time"
)

// Describes a loader entry stored in the database
type LoaderEntry struct {
	ID        float64   `json:"id"`        // Loader ID
	Name      string    `json:"name"`      // Loader name
	Key       string    `json:"key"`       // Loader key (only populated during creation)
	AgentName string    `json:"agentname"` // Loader environment, agent name
	LastSeen  time.Time `json:"lastseen"`  // Last time loader was used
	Enabled   bool      `json:"enabled"`   // Loader entry is active
}

func (le *LoaderEntry) Validate() error {
	return nil
}

func ValidateLoaderKey(key string) error {
	ok, err := regexp.MatchString("^[A-Za-z0-9]{1,256}$", key)
	if err != nil || !ok {
		return errors.New("loader key format is invalid")
	}
	return nil
}
