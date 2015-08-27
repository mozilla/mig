// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

/* The PGP package is a helper around Golang's OpenPGP implementation
 */
package pgp /* import "mig.ninja/mig/pgp" */

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"io"
	"strings"
)

// ArmoredKeysToKeyring takes a list of PGP keys in armored form and transforms
// it into a keyring that can be used in other openpgp's functions
func ArmoredKeysToKeyring(keys [][]byte) (keyring io.ReadSeeker, keycount int, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ArmoredPubKeysToKeyRing() -> %v", e)
		}
	}()
	var buf bytes.Buffer
	// iterate over the keys, and load them into an io.Reader keyring
	for i, key := range keys {
		// Load PGP public key
		el, err := openpgp.ReadArmoredKeyRing(bytes.NewBuffer(key))
		if err != nil {
			panic(fmt.Errorf("key num.%d failed to load with error %v", i, err))
		}
		keycount += 1
		if len(el) != 1 {
			err = fmt.Errorf("PGP key contains %d entities, wanted 1", len(el))
			panic(err)
		}
		// serialize entities into io.Reader
		if el[0].PrivateKey != nil {
			err = el[0].SerializePrivate(&buf, nil)
		} else {
			err = el[0].Serialize(&buf)
		}
		if err != nil {
			panic(err)
		}
	}
	keyring = bytes.NewReader(buf.Bytes())
	return
}

// KeyringToArmoredPubKeys reads all public keys from a keyring and returned their armored format
// into map of keys indexed by key fingerprint
func KeyringToArmoredPubKeys(keyring io.ReadCloser) (armoredkeys map[string][]byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("KeyringToArmoredPubKeys() -> %v", e)
		}
	}()
	els, err := openpgp.ReadArmoredKeyRing(keyring)
	if err != nil {
		panic(err)
	}
	for _, el := range els {
		fingerprint := hex.EncodeToString(el.PrimaryKey.Fingerprint[:])
		var pubkeybuf bytes.Buffer
		err = el.Serialize(&pubkeybuf)
		if err != nil {
			panic(err)
		}
		armoredbuf := bytes.NewBuffer(nil)
		ewrbuf, err := armor.Encode(armoredbuf, openpgp.PublicKeyType, nil)
		if err != nil {
			panic(err)
		}
		_, err = ewrbuf.Write(pubkeybuf.Bytes())
		if err != nil {
			panic(err)
		}
		ewrbuf.Close()
		armoredkeys[fingerprint] = armoredbuf.Bytes()
	}
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
	_, entity, err := Verify(data, signature, keyring)
	if err != nil {
		panic(err)
	}
	fingerprint = hex.EncodeToString(entity.PrimaryKey.Fingerprint[:])
	return
}

func GenerateKeyPair(name, desc, email string) (pubkey, privkey []byte, fp string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GenerateKeyPair() -> %v", e)
		}
	}()
	// generate a private key
	ent, err := openpgp.NewEntity(name, desc, email, nil)
	if err != nil {
		panic(err)
	}
	// serialize the private key
	pkbuf := bytes.NewBuffer(nil)
	err = ent.SerializePrivate(pkbuf, nil)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(nil)
	ewrbuf, err := armor.Encode(buf, openpgp.PrivateKeyType, nil)
	if err != nil {
		panic(err)
	}
	_, err = ewrbuf.Write(pkbuf.Bytes())
	if err != nil {
		panic(err)
	}
	ewrbuf.Close()
	privkey = buf.Bytes()
	// serialize the public key
	pkbuf = bytes.NewBuffer(nil)
	err = ent.Serialize(pkbuf)
	if err != nil {
		panic(err)
	}
	buf = bytes.NewBuffer(nil)
	ewrbuf, err = armor.Encode(buf, openpgp.PublicKeyType, nil)
	if err != nil {
		panic(err)
	}
	_, err = ewrbuf.Write(pkbuf.Bytes())
	if err != nil {
		panic(err)
	}
	ewrbuf.Close()
	pubkey = buf.Bytes()
	// validate the public key and obtain a fingerprint from it
	fp, err = LoadArmoredPubKey(pubkey)
	if err != nil {
		panic(err)
	}
	return
}

func ArmorPubKey(pubkey []byte) (armoredPubKey []byte, err error) {
	var pubkeybuf bytes.Buffer
	// Load PGP public key
	el, err := openpgp.ReadArmoredKeyRing(bytes.NewBuffer(pubkey))
	if err != nil {
		panic(err)
	}
	// serialize entities into io.Reader
	err = el[0].Serialize(&pubkeybuf)
	if err != nil {
		panic(err)
	}
	armoredbuf := bytes.NewBuffer(nil)
	ewrbuf, err := armor.Encode(armoredbuf, openpgp.PublicKeyType, nil)
	if err != nil {
		panic(err)
	}
	_, err = ewrbuf.Write(pubkeybuf.Bytes())
	if err != nil {
		panic(err)
	}
	ewrbuf.Close()
	armoredPubKey = armoredbuf.Bytes()
	return
}
