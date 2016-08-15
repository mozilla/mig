// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import (
	"fmt"
	"strings"
	"time"
)

type Investigator struct {
	ID             float64   `json:"id,omitempty"`
	Name           string    `json:"name"`
	PGPFingerprint string    `json:"pgpfingerprint"`
	PublicKey      []byte    `json:"publickey,omitempty"`
	PrivateKey     []byte    `json:"privatekey,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdat"`
	LastModified   time.Time `json:"lastmodified"`
	Permissions    int       `json:"permissions"`
}

const (
	StatusActiveInvestigator   string = "active"
	StatusDisabledInvestigator string = "disabled"
)

// Permissions that can be assigned to investigators
const (
	PermLoaders   = 1 << iota // Create and manage loaders
	PermManifests             // Create and manage manifests
	PermAdmin                 // Create and manage investigators
)

type InvestigatorPermText struct {
	Text  string
	Value int
}

// InvestigatorPermissions maps permission string values to the actual
// permission values that can be set in the bitmask; used primarily for
// display purposes within UI
//
// By default, an investigator has no additional permissions which means
// the investigator can run investigations, review action results, etc.
// If additional permissions are applied, this gives the investigator
// access to additional endpoints related to administration of the MIG
// platform.
var InvestigatorPermissions = []InvestigatorPermText{
	{Text: "PermLoaders", Value: PermLoaders},
	{Text: "PermManifests", Value: PermManifests},
	{Text: "PermAdmin", Value: PermAdmin},
}

// Return a value representing all possible permissions an investigator
// can have
func InvestigatorPermsAll() int {
	return PermLoaders | PermManifests | PermAdmin
}

// Process a list of permission string values and return a permission
// bitmask
func InvestigatorPermsFromStrings(slist []string) (int, error) {
	ret := 0
	for _, x := range slist {
		found := false
		for _, y := range InvestigatorPermissions {
			if strings.ToLower(y.Text) == strings.ToLower(x) {
				ret |= y.Value
				found = true
			}
		}
		if !found {
			return 0, fmt.Errorf("Invalid permission %v", x)
		}
	}
	return ret, nil
}
