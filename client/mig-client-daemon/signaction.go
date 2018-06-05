// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path"

	"mig.ninja/mig"
	"mig.ninja/mig/pgp"
)

func main() {
	var action mig.Action

	secretKeyID := flag.String("key", "", "Fingerprint of secret key to use for signing")
	passphrase := flag.String("passphrase", "", "Passphrase for secret key to use for signing")
	secringPath := flag.String("secring", defaultSecringPath(), "Path to secring.gpg file")

	flag.Parse()

	if secretKeyID == nil || *secretKeyID == "" {
		fmt.Println("Missing key ID")
		return
	}
	if passphrase == nil || *passphrase == "" {
		fmt.Println("Missing secret key passphrase")
		return
	}
	if secringPath == nil || *secringPath == "" {
		fmt.Println("Missing secring.gpg path")
		return
	}

	decoder := json.NewDecoder(os.Stdin)
	decodeErr := decoder.Decode(&action)
	if decodeErr != nil {
		panic(decodeErr)
	}

	pgp.CachePassphrase(*passphrase)

	secring, err := os.Open(*secringPath)
	if err != nil {
		panic(err)
	}

	signature, signErr := action.Sign(*secretKeyID, secring)
	if signErr != nil {
		panic(signErr)
	}

	fmt.Println(signature)
}

func defaultSecringPath() string {
	curUser, err := user.Current()
	if err != nil {
		return ""
	}
	return path.Join(curUser.HomeDir, ".gnupg", "secring.gpg")
}
