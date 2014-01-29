/* Mozilla InvestiGator PGP

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package verify

import (
	"bytes"
	"code.google.com/p/go.crypto/openpgp"
	"code.google.com/p/go.crypto/openpgp/armor"
	"fmt"
	"io"
	"strings"
)

// Verify() checks the validity of a signature for some data,
// and returns a boolean set to true if valid and an OpenPGP Entity
func Verify(data string, signature string, keyring io.Reader) (valid bool, entity *openpgp.Entity, err error) {
	valid = false

	// re-armor signature and transform into io.Reader
	sigReader := strings.NewReader(reArmor(signature))

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

// reArmor takes a single line armor and turns it back into an PGP-style
// multi-line armored string (thank you, camlistore folks)
func reArmor(line string) string {
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
		if chunkLen > 60 {
			chunkLen = 60
		}
		fmt.Fprintf(buf, "%s\n", payload[0:chunkLen])
		payload = payload[chunkLen:]
	}
	fmt.Fprintf(buf, "%s\n-----BEGIN PGP SIGNATURE-----\n", crc)
	return buf.String()
}
