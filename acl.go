// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import (
	"fmt"
	"strings"
)

type ACL []Permission

type Permission map[string]struct {
	MinimumWeight int
	Investigators map[string]struct {
		Fingerprint string
		Weight      int
	}
}

// verifyPermission controls that the PGP keys, identified by their fingerprints, that
// signed an operation are sufficient to allow this operation to run
func verifyPermission(operation Operation, permName string, perm Permission, fingerprints []string) (err error) {
	if perm[permName].MinimumWeight < 1 {
		return fmt.Errorf("Invalid permission '%s'. Must require at least 1 signature, has %d",
			permName, perm[permName].MinimumWeight)
	}
	var seenFp []string
	signaturesWeight := 0
	for _, fp := range fingerprints {
		// if the same key is used to sign multiple times, return an error
		for _, seen := range seenFp {
			if seen == fp {
				return fmt.Errorf("Permission violation: key id '%s' used to sign multiple times.", fp)
			}
		}
		for _, signer := range perm[permName].Investigators {
			if strings.ToUpper(fp) == strings.ToUpper(signer.Fingerprint) {
				signaturesWeight += signer.Weight
			}
		}
		seenFp = append(seenFp, fp)
	}
	if signaturesWeight < perm[permName].MinimumWeight {
		return fmt.Errorf("Permission denied for operation '%s'. Insufficient signatures weight. Need %d, got %d",
			operation.Module, perm[permName].MinimumWeight, signaturesWeight)
	}
	return
}
