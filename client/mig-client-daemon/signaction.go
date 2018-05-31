// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"mig.ninja/mig"
	"mig.ninja/mig/pgp"
)

const DANGEROUS_PASSPHRASE string = "KEY PASSPHRASE (or empty string)"
const keyID string = "KEY FINGERPRINT"

const secringPath string = "/PATH TO HOME/.gnupg/secring.gpg"

const toSign string = `{"id": 0, "name": "b2b943", "target": "tags->>'operator'='IT'", "description": {}, "threat": {}, "validfrom": "0001-01-01T00:00:00Z", "expireafter": "2018-05-18T17:10:42.656564731-04:00", "operations": [{"module": "pkg", "parameters": {"pkgmatch": {"matches": ["*libssl*"]}, "vermatch": ""}}], "pgpsignatures": null, "starttime": "0001-01-01T00:00:00Z", "finishtime": "0001-01-01T00:00:00Z", "lastupdatetime": "0001-01-01T00:00:00Z", "counters": {}, "syntaxversion": 2}`

func main() {
	var action mig.Action

	decoder := json.NewDecoder(strings.NewReader(toSign))
	decodeErr := decoder.Decode(&action)
	if decodeErr != nil {
		panic(decodeErr)
	}

	pgp.CachePassphrase(DANGEROUS_PASSPHRASE)

	secring, err := os.Open(secringPath)
	if err != nil {
		panic(err)
	}

	signature, signErr := action.Sign(keyID, secring)
	if signErr != nil {
		panic(signErr)
	}

	fmt.Println("Signature\n")
	fmt.Println(signature)
}
