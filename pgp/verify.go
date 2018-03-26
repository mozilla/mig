// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package pgp /* import "mig.ninja/mig/pgp" */

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"io"
	"strings"
)

// Verify() checks the validity of a signature for some data,
// and returns a boolean set to true if valid and an OpenPGP Entity
func Verify(data string, signature string, keyring io.Reader) (valid bool, entity *openpgp.Entity, err error) {
	valid = false

	// re-armor signature and transform into io.Reader
	sig := reArmorSignature(signature)
	sigReader := strings.NewReader(sig)

	// decode armor
	sigBlock, err := armor.Decode(sigReader)
	if err != nil {
		panic(err)
	}
	if sigBlock.Type != "PGP SIGNATURE" {
		err = fmt.Errorf("Wrong signature type '%s'", sigBlock.Type)
		panic(err)
	}

	// convert to io.Reader
	srcReader := strings.NewReader(data)

	// open the keyring
	ring, err := openpgp.ReadKeyRing(keyring)
	if err != nil {
		panic(err)
	}

	entity, err = openpgp.CheckDetachedSignature(ring, srcReader, sigBlock.Body)
	if err != nil {
		panic(err)
	}

	// we passed, signature is valid
	valid = true

	return
}

// reArmorSignature takes a single line armor and turns it back into an PGP-style
// multi-line armored string (thank you, camlistore folks)
func reArmorSignature(line string) string {
	lastEq := strings.LastIndex(line, "=")
	if lastEq == -1 {
		return ""
	}
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "-----BEGIN PGP SIGNATURE-----\n\n")
	payload := line[0:lastEq]
	crc := line[lastEq:]
	for len(payload) > 0 {
		chunkLen := len(payload)
		if chunkLen > 64 {
			chunkLen = 64
		}
		fmt.Fprintf(buf, "%s\n", payload[0:chunkLen])
		payload = payload[chunkLen:]
	}
	fmt.Fprintf(buf, "%s\n-----END PGP SIGNATURE-----", crc)
	return buf.String()
}
