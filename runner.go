// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package mig /* import "mig.ninja/mig" */

// Describes results that are produced by mig-runner
type RunnerResult struct {
	Action     Action    `json:"action"`
	Commands   []Command `json:"commands"`
	EntityName string    `json:"name"`
	UsePlugin  string    `json:"plugin"`
}
