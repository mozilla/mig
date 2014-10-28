// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package pgp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"mig/pgp/verify"
	"strings"

	"code.google.com/p/go.crypto/openpgp"
)

// ArmoredPubKeysToKeyring takes a list of public PGP key in armored form and transforms
// it into a keyring that can be used in other openpgp's functions
func ArmoredPubKeysToKeyring(pubkeys []string) (keyring io.ReadSeeker, keycount int, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ArmoredPubKeysToKeyRing() -> %v", e)
		}
	}()
	var buf bytes.Buffer
	// iterate over the keys, and load them into an io.Reader keyring
	for _, key := range pubkeys {
		// Load PGP public key
		el, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(key))
		if err != nil {
			panic(err)
		}
		keycount += 1
		if len(el) != 1 {
			err = fmt.Errorf("Public GPG Key contains %d entities, wanted 1", len(el))
			panic(err)
		}
		// serialize entities into io.Reader
		err = el[0].Serialize(&buf)
		if err != nil {
			panic(err)
		}
	}
	keyring = bytes.NewReader(buf.Bytes())
	return
}

// LoadArmoredPubKey takes a single public key as a byte slice, validates it, and returns its
// its fingerprint or an error
func LoadArmoredPubKey(pubkey []byte) (pgpfingerprint string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("LoadArmoredPubKey() -> %v", e)
		}
	}()
	el, err := openpgp.ReadArmoredKeyRing(bytes.NewBuffer(pubkey))
	if err != nil {
		panic(err)
	}
	if len(el) != 1 {
		err = fmt.Errorf("Public GPG Key contains %d entities, wanted 1", len(el))
		panic(err)
	}
	entity := el[0]
	fp := hex.EncodeToString(entity.PrimaryKey.Fingerprint[:])
	pgpfingerprint = strings.ToUpper(fp)
	return
}

func GetFingerprintFromSignature(data string, signature string, keyring io.Reader) (fingerprint string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetFingerprintFromSignature() -> %v", e)
		}
	}()
	_, entity, err := verify.Verify(data, signature, keyring)
	if err != nil {
		panic(err)
	}
	fingerprint = hex.EncodeToString(entity.PrimaryKey.Fingerprint[:])
	return
}
