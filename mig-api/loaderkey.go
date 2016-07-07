// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

// Contains functions related to loader key authentication, including
// key hashing and key comparisons

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"mig.ninja/mig"
)

func hashLoaderKey(key string, salt []byte) (ret string, err error) {
	if salt == nil {
		salt = make([]byte, 16)
		_, err = rand.Read(salt)
		if err != nil {
			return
		}
	}
	hv := pbkdf2.Key([]byte(key), salt, 4096, 32, sha256.New)
	return fmt.Sprintf("%x%x", salt, hv), nil
}

func hashAuthenticateLoader(lkey string) (ldr mig.LoaderEntry, err error) {
	err = mig.ValidateLoaderPrefixAndKey(lkey)
	if err != nil {
		return
	}
	prefix := lkey[:mig.LoaderPrefixLength]
	suppkey := lkey[mig.LoaderPrefixLength:]
	id, hashkey, err := ctx.DB.GetLoaderIDAndKey(prefix)
	if err != nil {
		return
	}
	// Extract the 16 byte (32 char hex encoded) salt
	salt, err := hex.DecodeString(hashkey[:32])
	if err != nil {
		return
	}
	tryhash, err := hashLoaderKey(suppkey, salt)
	if err != nil {
		return
	}
	if tryhash != hashkey {
		err = fmt.Errorf("Loader key authentication failed")
		return
	}
	ldr, err = ctx.DB.GetLoaderFromID(id)
	return
}
