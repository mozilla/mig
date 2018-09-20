// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly <zmullaly@mozilla.com>

package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

// PGPAuth implements CryptoAuth to provide crypto services using PGP.
type PGPAuth struct {
	secring          *io.ReadSeeker
	cachedPassphrase string
}

// NewPGPAuth constructs a new PGPAuth instance that will load keys
// from a secring given by its path.
func NewPGPAuth(secring *io.ReadSeeker, passphrase string) PGPAuth {
	return PGPAuth{
		secring:          secring,
		cachedPassphrase: "",
	}
}

// Identifier specifies which implementation of CryptoAuth this is.
func (pgp *PGPAuth) Identifier() AuthSchemeId {
	return AuthSchemePGP
}

// Sign creates a signature over a message using a particular key.
func (pgp *PGPAuth) Sign(key Fingerprint, message []byte) ([]byte, error) {
	pgp.secring.Seek(0, 0)

	keyring, err := openpgp.ReadKeyRing(pgp.secring)
	if err != nil {
		return []byte{}, err
	}
	var signer *openpgp.Entity
	found := false
	for _, entity := range keyring {
		if entity.PrivateKey == nil {
			return fmt.Errorf("secring contains an entity without private key data")
		}
		fingerprint := strings.ToUpper(
			hex.EncodeToString(entity.PrivateKey.PublicKey.Fingerprint[:]))
		if key == fingerprint {
			signer = entity
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Signer '%s' not found", key)
	}
	// If the private key is encrypted, try to decrypt it with a cached passphrase
	// then try with an agent or by asking the user for a passphrase.
	if signer.PrivateKey.Encrypted {
		err = signer.PrivateKey.Decrypt([]byte(pgp.cachedPassphrase))
		if err != nil {
			var pass string
			signer, pass, err = decryptEntity(signer)
			if err != nil {
				return []byte{}, err
			}
			if pass != "" {
				cachedPassphrase = pass
			}
		}
	}
	out := bytes.NewBuffer(nil)
	msg := bytes.NewBuffer(message)
	err = openpgp.ArmoredDetachSign(out, signer, msg, nil)
	if err != nil {
		return fmt.Errorf("Signing failed: '%s'", err.Error())
	}
	sig, err := deArmor(out.String())
	if err != nil {
		return fmt.Errorf("Error converting signature to string: '%s'", err.Error())
	}
	decodedSig, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return fmt.Errorf("Error decoding signature from base64: '%s'", err.Error())
	}
	return sig, nil
}

// Verify determines if a signature over a message is valid.  If the signature
// is valid, nil will be returned.
func (pgp *PGPAuth) Verify(key Fingerprint, signature, message []byte) error {
	var err error

	pgp.secring.Seek(0, 0)

	sig := reArmorSignature(base64.StdEncoding.EncodeToString(signature))
	sigReader := strings.NewReader(sig)

	sigBlock, err := armor.Decode(sigReader)
	if err != nil {
		return err
	}
	if sigBlock.Type != "PGP SIGNATURE" {
		return fmt.Errorf("Wrong signature type '%s'", sigBlock.Type)
	}
	srcReader := bytes.NewReader(message)
	ring, err := openpgp.ReadKeyRing(pgp.keyring)
	if err != nil {
		return err
	}
	_, err := openpgp.CheckDetachedSignature(ring, srcReader, sigBlock.Body)
	if err != nil {
		return err
	}
	return nil
}

// decryptEntity calls gnupg-agent and pinentry to obtain a passphrase and
// decrypt the private key of a given entity (thank you, camlistore folks)
func decryptEntity(s *openpgp.Entity) (ds *openpgp.Entity, pass string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("pgp.decryptEntity(): %v", e)
		}
	}()
	ds = s
	// TODO: syscall.Mlock a region and keep pass phrase in it.
	pubk := &ds.PrivateKey.PublicKey
	desc := fmt.Sprintf("Need to unlock GPG key %s to use it for signing.",
		pubk.KeyIdShortString())

	conn, err := gpgagent.NewConn()
	switch err {
	case gpgagent.ErrNoAgent:
		fmt.Fprintf(os.Stderr, "Note: gpg-agent not found; resorting to on-demand password entry.\n")
	case nil:
		defer conn.Close()
		req := &gpgagent.PassphraseRequest{
			CacheKey: "mig:pgpsign:" + pubk.KeyIdShortString(),
			Prompt:   "Passphrase",
			Desc:     desc,
		}
		for tries := 0; tries < 3; tries++ {
			pass, err := conn.GetPassphrase(req)
			if err == nil {
				err = ds.PrivateKey.Decrypt([]byte(pass))
				if err == nil {
					return ds, pass, err
				}
				req.Error = "Passphrase failed to decrypt: " + err.Error()
				conn.RemoveFromCache(req.CacheKey)
				continue
			}
			if err == gpgagent.ErrCancel {
				panic("failed to decrypt key; action canceled")
			}
		}
	default:
		panic(err)
	}

	pinReq := &pinentry.Request{Desc: desc, Prompt: "Passphrase"}
	for tries := 0; tries < 3; tries++ {
		pass, err = pinReq.GetPIN()
		if err == nil {

			err = ds.PrivateKey.Decrypt([]byte(pass))
			if err == nil {
				return ds, pass, err
			}
			pinReq.Error = "Passphrase failed to decrypt: " + err.Error()
			continue
		}
		if err == pinentry.ErrCancel {
			panic("failed to decrypt key; action canceled")
		}
	}
	return ds, "", fmt.Errorf("decryptEntity(): failed to decrypt key %q: %v", pubk.KeyIdShortString(), err)
}

// deArmor takes a multi line armored GPG signature, and turns it back
// into a single line signature (thank you, camlistore folks)
func deArmor(sig string) (str string, err error) {
	index1 := strings.Index(sig, "\n\n")
	index2 := strings.Index(sig, "\n-----")
	if index1 == -1 || index2 == -1 {
		err = fmt.Errorf("Failed to parse signature from gpg.")
		return
	}
	inner := sig[index1+2 : index2]
	str = strings.Replace(inner, "\n", "", -1)
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
