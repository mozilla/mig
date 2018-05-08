// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

const migAPIVersion int = 1
const pgpAuthHeader string = "X-PGPAUTHORIZATION"

const emptyToken Token = Token{
	token: "",
}

// UnsignedToken is the base of a token that a signature can be provided for
// and then uploaded into an `Authenticator`.
type UnsignedToken struct {
	token string
}

// Token contains a signed token string passed to the MIG API to prove that
// the sender is an investigator known by the API.
type Token struct {
	token string
}

// PGPAuthorization is an `Authenticator` that assists in performing
// authentication in two steps.
// The first step is to generate an unsigned token containing information
// required by the API.
// The second step is to update the token with a signature provided by the
// investigator so that it can be used to authenticate requests.
type PGPAuthorization struct {
	token Token
}

// NewPGPAuthorization constructs a new `PGPAuthorization`.
func NewPGPAuthorization() PGPAuthorization {
	return PGPAuthorization{
		token: emptyToken,
	}
}

// ProvideSignature appends a signature to an unsigned token so that it can be
// used to make requests to the MIG API.
func (tkn UnsignedToken) ProvideSignature(signature string) Token {
	return Token{
		token: tkn.token + ";" + signature,
	}
}

// String produces the string representation of a signed PGPAUTHORIZATION token.
func (tkn Token) String() string {
	return tkn.token
}

// GenerateUnsignedToken creates and records an unsigned token that the
// `PGPAuthorization` can receive an update for containing a signature.
func (auth *PGPAuthorization) GenerateUnsignedToken() UnsignedToken {
	max := big.NewInt(0x0FFFFFFFFFFFFFFD)
	nonce, err := rand.Int(rand.Reader, max)

	for err != nil {
		nonce, err = rand.Int(rand.Reader, nil)
	}

	currentTime := time.Now().UTC().String()

	return UnsignedToken{
		token: fmt.Sprintf("%d;%s;%s", migAPIVersion, currentTime, nonce.String()),
	}
}

// StoreSignedToken records a token with a signature provided so that it can be
// used by `PGPAuthorization`
func (auth *PGPAuthorization) StoreSignedToken(token Token) {
	auth.token = token
}

func (auth PGPAuthorization) Authenticate(req *http.Request) error {
	if auth.token == emptyToken {
		return errors.New("PGPAuthorization cannot perform authorization before a signed token is set.")
	}

	req.Header.Set(pgpAuthHeader, auth.token.String())
	return nil
}
