// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package ident

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// EmptyID can be used to indicate that an ID was not created.
const EmptyID Identifier = Identifier("")

// Identifier contains a unique identifier for a resource managed by
// the client daemon.
type Identifier string

// UniquenessTest functions provide a guarantee of the uniqueness of an
// identifier, dependent on whatever context the identifier is used in.
type UniquenessTest func(Identifier) bool

// GenerateUniqueID creates an identifier for a resource managed by the
// client daemon with a guarantee of uniqueness.
func GenerateUniqueID(
	bytesToGenerate uint,
	sleepBetweenReadAttempts time.Duration,
	isUnique UniquenessTest,
) Identifier {
	randBytes := make([]byte, bytesToGenerate)

	for {
		// We don't necessarily need cryptographically secure random bytes for IDs
		// but they're reliable and easy to deal with.
		bytesRead, err := rand.Read(randBytes)
		if err != nil || uint(bytesRead) < bytesToGenerate {
			// If we encountered an error, it's probablt because the OS' pool of
			// entropy has been exhausted.  So we will just wait a little bit.
			<-time.After(sleepBetweenReadAttempts)
			continue
		}

		identifier := Identifier(hex.EncodeToString(randBytes))

		if !isUnique(identifier) {
			continue
		}

		// We have generated a random ID that is not already in use.
		return identifier
	}
}
