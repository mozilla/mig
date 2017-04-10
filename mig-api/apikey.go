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

const APIKeyLength = 32
const APIHashedKeyLength = 32
const APISaltLength = 16

func hashAPIKey(key string, salt []byte, keylen int, saltlen int) (ret []byte, retsalt []byte, err error) {
	if key == "" {
		err = fmt.Errorf("loader key cannot be zero length")
		return
	}
	if salt == nil {
		retsalt = make([]byte, saltlen)
		_, err = rand.Read(retsalt)
		if err != nil {
			return
		}
	} else {
		retsalt = salt
	}
	ret = pbkdf2.Key([]byte(key), retsalt, 4096, keylen, sha256.New)
	return ret, retsalt, nil
}

// Verify an X-MIGAPIKEY header, if the supplied header value matches any key
// configured for an investigator in the database, the investigator is returned,
// otherwise an error is returned.
func verifyAPIKey(key string) (inv mig.Investigator, err error) {
	reterr := fmt.Errorf("API key authentication failed")
	apiinvs, err := ctx.DB.InvestigatorAPIKeyAuthHelpers()
	if err != nil {
		return inv, reterr
	}
	for _, x := range apiinvs {
		tryhash, _, err := hashAPIKey(key, x.Salt, len(x.APIKey), len(x.Salt))
		if err != nil {
			return inv, reterr
		}
		if bytes.Equal(tryhash, x.APIKey) {
			inv, err = ctx.DB.InvestigatorByID(x.ID)
			if err != nil {
				return inv, reterr
			}
			return inv, nil
		}
	}
	return inv, reterr
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
	tryhash, _, err := hashAPIKey(suppkey, lad.Salt, mig.LoaderHashedKeyLength, mig.LoaderSaltLength)
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
