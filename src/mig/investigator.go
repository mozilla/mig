// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package mig

import (
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
}

const (
	StatusActiveInvestigator   string = "active"
	StatusDisabledInvestigator string = "disabled"
)
