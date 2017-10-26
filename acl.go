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

// ACL defines an access control list used by the agent to determine what investigators
// can call a given module. The key in this map is the module name, and can be "default" in
// which case this element will be used if no key for a given module exists.
//
// The value includes a minimum weight to authorize the request, and a map of investigators
// with the key of the map being the name of the investigator, and the value storing the PGP
// fingerprint of the investigators key and the weight that investigator has.
type ACL map[string]struct {
	MinimumWeight int
	Investigators map[string]struct {
		Fingerprint string
		Weight      int
	}
}

// verifyPermission validates that the fingerprints are permitted to execute an operation by
// comparing the fingerprints to the fingerprints and weight specifications in ACL acl.
func verifyPermission(moduleName string, acl ACL, fingerprints []string) error {
	aclname := moduleName
	aclent, ok := acl[moduleName]
	if !ok {
		// No ACL entry found for this module name, see if we can find a default
		aclent, ok = acl["default"]
		if !ok {
			return fmt.Errorf("no ACL entry found for %v, and no default present", moduleName)
		}
		aclname = "default"
	}
	if aclent.MinimumWeight < 1 {
		return fmt.Errorf("invalid ACL %v, weight must be > 0", aclname)
	}
	var seenFp []string
	signaturesWeight := 0
	for _, fp := range fingerprints {
		// if the same key is used to sign multiple times, return an error
		for _, seen := range seenFp {
			if seen == fp {
				return fmt.Errorf("permission violation: key %v used to sign multiple times", fp)
			}
		}
		for _, signer := range aclent.Investigators {
			if strings.ToUpper(fp) == strings.ToUpper(signer.Fingerprint) {
				signaturesWeight += signer.Weight
			}
		}
		seenFp = append(seenFp, fp)
	}
	if signaturesWeight < aclent.MinimumWeight {
		return fmt.Errorf("permission denied for operation %v, insufficient signatures weight"+
			" (need %v, got %v)", moduleName, aclent.MinimumWeight, signaturesWeight)
	}
	return nil
}
