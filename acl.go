// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package mig

import (
	"fmt"
	"io"
	"time"

	"mig.ninja/mig/pgp"
)

type ACL struct {
	Name          string       `json:"name"`
	Target        string       `json:"target"`
	ValidFrom     time.Time    `json:"validfrom"`
	ExpireAfter   time.Time    `json:"expireafter"`
	Permissions   []Permission `json:"permissions"`
	PGPSignatures []string     `json:"pgpsignatures"`
}

type Permission struct {
	Module        string             `json:"module"`
	MinimumWeight float64            `json:"minimumweight"`
	Investigators []InvestigatorPerm `json:"investigators"`
}

type InvestigatorPerm struct {
	ID             float64 `json:"id"`
	Name           string  `json:"name"`
	PGPFingerprint string  `json:"pgpfingerprint"`
	Weight         float64 `json:"weight"`
}

//  concatenates Action components into a string
func (acl ACL) String() string {
	var permstr string
	for _, perm := range acl.Permissions {
		permstr += fmt.Sprintf("mod=%s;minweight=%.0f;inv=[",
			perm.Module, perm.MinimumWeight)
		for _, inv := range perm.Investigators {
			permstr += fmt.Sprintf("[id=%.0f;name=%s;fp=%s;weight=%.0f];",
				inv.ID, inv.Name, inv.PGPFingerprint, inv.Weight)
		}
		permstr += "];"
	}
	return fmt.Sprintf("name=%s;target=%s;validfrom=%d;expireafter=%d;permissions=[%s];",
		acl.Name, acl.Target, acl.ValidFrom.UTC().Unix(), acl.ExpireAfter.UTC().Unix(), permstr)
}

// Sign computes a PGP signature for a given ACL and returns it as a string
func (acl ACL) Sign(keyid string, secring io.Reader) (sig string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Sign() -> %v", e)
		}
	}()
	sig, err = pgp.Sign(acl.String(), keyid, secring)
	if err != nil {
		panic(err)
	}
	return
}

// VerifySignatures verifies that the Action contains valid signatures from
// known investigators. It does not verify permissions.
func (acl ACL) VerifySignatures(reqsig int, keyring io.ReadSeeker) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("VerifySignatures() -> %v", e)
		}
	}()
	var seenFp []string
	aclstr := acl.String()
	for _, sig := range acl.PGPSignatures {
		// rewind the keyring
		_, err = keyring.Seek(0, 0)
		if err != nil {
			panic(err)
		}
		valid, entity, err := pgp.Verify(aclstr, sig, keyring)
		if err != nil {
			panic(fmt.Errorf("Failed to verify PGP Signature: %v", err))
		}
		if !valid {
			panic("Invalid PGP Signature")
		}
		fingerprint := pgp.GetFingerprintFromEntity(entity)
		for _, seen := range seenFp {
			if seen == fingerprint {
				panic(fmt.Errorf("ACL violation: key id '%s' used to sign multiple times.", fingerprint))
			}
		}
		seenFp = append(seenFp, fingerprint)
	}
	if len(seenFp) < reqsig {
		panic(fmt.Errorf("Not enough signatures. Needed %d, got %d", reqsig, len(seenFp)))
	}
	return
}

// verifyPermission controls that the PGP keys, identified by their fingerprints, that
// signed an operation are sufficient to allow this operation to run
//func verifyPermission(operation Operation, permName string, perm Permission, fingerprints []string) (err error) {
//	if perm[permName].MinimumWeight < 1 {
//		return fmt.Errorf("Invalid permission '%s'. Must require at least 1 signature, has %d",
//			permName, perm[permName].MinimumWeight)
//	}
//	var seenFp []string
//	signaturesWeight := 0
//	for _, fp := range fingerprints {
//		// if the same key is used to sign multiple times, return an error
//		for _, seen := range seenFp {
//			if seen == fp {
//				return fmt.Errorf("Permission violation: key id '%s' used to sign multiple times.", fp)
//			}
//		}
//		for _, signer := range perm[permName].Investigators {
//			if strings.ToUpper(fp) == strings.ToUpper(signer.Fingerprint) {
//				signaturesWeight += signer.Weight
//			}
//		}
//		seenFp = append(seenFp, fp)
//	}
//	if signaturesWeight < perm[permName].MinimumWeight {
//		return fmt.Errorf("Permission denied for operation '%s'. Insufficient signatures weight. Need %d, got %d",
//			operation.Module, perm[permName].MinimumWeight, signaturesWeight)
//	}
//	return
//}
