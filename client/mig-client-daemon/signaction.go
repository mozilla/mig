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

	"mig.ninja/mig"
	"mig.ninja/mig/pgp"
)

// CONFIGURATION
const (
	DANGEROUS_PASSPHRASE = "PASSPHRASE FOR SECRET KEY"
	keyID                = "SECRET KEY FINGERPRINT"
	secringPath          = "/HOME/.gnupg/secring.gpg"
)

func main() {
	var action mig.Action

	decoder := json.NewDecoder(os.Stdin)
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

	fmt.Println(signature)
}
