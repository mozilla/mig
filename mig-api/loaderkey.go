// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

// Contains functions related to loader key authentication, including
// key hashing and key comparisons

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"mig.ninja/mig"
)

func hashLoaderKey(key string, salt []byte) (ret []byte, retsalt []byte, err error) {
	if salt == nil {
		retsalt = make([]byte, mig.LoaderSaltLength)
		_, err = rand.Read(retsalt)
		if err != nil {
			return
		}
	} else {
		retsalt = salt
	}
	ret = pbkdf2.Key([]byte(key), retsalt, 4096, mig.LoaderHashedKeyLength, sha256.New)
	return ret, retsalt, nil
}

func hashAuthenticateLoader(lkey string) (ldr mig.LoaderEntry, err error) {
	err = mig.ValidateLoaderPrefixAndKey(lkey)
	if err != nil {
		return
	}
	prefix := lkey[:mig.LoaderPrefixLength]
	suppkey := lkey[mig.LoaderPrefixLength:]
	lad, err := ctx.DB.GetLoaderAuthDetails(prefix)
	if err != nil {
		return
	}
	tryhash, _, err := hashLoaderKey(suppkey, lad.Salt)
	if err != nil {
		return
	}
	if !bytes.Equal(tryhash, lad.Hash) {
		err = fmt.Errorf("Loader key authentication failed")
		return
	}
	ldr, err = ctx.DB.GetLoaderFromID(lad.ID)
	return
}
