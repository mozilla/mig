// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly <zmullaly@mozilla.com>

package auth

// AuthSchemeId is used to identify authentication schemes supported by MIG.
type AuthSchemeId string

// Fingerprint is a details-agnostic identifier for a key.
// Implementations of `CryptoAuth` will store keys and which one should be used
// to produce or verify a signature can be specified by indicating the fingerprint
// of the key to use.
type Fingerprint string

const (
	AuthSchemePGP     AuthSchemeId = "auth-scheme-pgp"
	AuthSchemeEd25519 AuthSchemeId = "auth-scheme-ed25519"
)

// CryptoAuth implementations provide the following services:
// 1. Self-identification for runtime selection and inspection,
// 2. Public and private key management,
// 3. Signing messages with public keys and
// 4. Signature verification using private keys.
type CryptoAuth interface {
	// Identifier retrieves what should be a unique ID associated with the implementation.
	Identifier() AuthSchemeId

	// Sign will attempt to produce a signature over a message with a public key managed
	// by the implementation.
	Sign(Fingerprint, []byte) ([]byte, error)

	// Verify will attempt to use a private key identified by its fingerprint to verify a
	// signature over a message.
	// A return value of `nil` indicates that the signature is valid.
	Verify(Fingerprint, []byte, []byte) error
}

// TryMultipleKeyVerification will attempt to verify a signature over a message using an
// auth scheme and multiple keys.
// As with `CryptoAuth.Verify`, this function will return an error value of `nil` if one
// of the designated keys successfully validates the signature. It will also return that
// key's fingerprint.
func TryMultipleKeyVerification(
	auth CryptoAuth,
	keys []Fingerprint,
	signature []byte,
	message []byte,
) (Fingerprint, error) {
	var err error

	for _, fingerprint := range keys {
		err = auth.Verify(fingerprint, signature, message)
		if err == nil {
			return fingerprint, nil
		}
	}
	return Fingerprint(""), err
}

// Supported can be used to verify that an `AuthSchemeId` is known by MIG
// rather than having been constructed accidentally by calling `AuthSchemeId(someString)`.
func (scheme AuthSchemeId) Supported() bool {
	supported := []AuthSchemeId{
		AuthSchemePGP,
		AuthSchemeEd25519,
	}
	for _, sup := range supported {
		if scheme == sup {
			return true
		}
	}
	return false
}
