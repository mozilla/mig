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
	"sync"
	"time"
)

const migAPIVersion int = 1
const pgpAuthHeader string = "X-PGPAUTHORIZATION"

// Challenge is the base of a token that a signature can be provided for
// and then loaded into an `Authenticator`.
type Challenge struct {
	challenge string
}

// Token contains a signed token string passed to the MIG API to prove that
// the sender is an investigator known by the API.
type Token struct {
	token string
}

// PGPAuthorizer is an `Authenticator` that enables PGP-based authentication to
// the MIG API.
type PGPAuthorizer struct {
	token Token
	lock  *sync.Mutex
}

func emptyToken() Token {
	return Token{
		token: "",
	}
}

// NewPGPAuthorizer constructs a `PGPAuthorizer` that
func NewPGPAuthorizer() PGPAuthorizer {
	return PGPAuthorizer{
		token: emptyToken(),
		lock:  new(sync.Mutex),
	}
}

// NewChallenge constructs a `Challenge` out of a previously-issued challenge
// as a string.
func NewChallenge(chalString string) Challenge {
	return Challenge{
		challenge: chalString,
	}
}

// String returns the string representation of the PGP challenge.
func (ch Challenge) String() string {
	return ch.challenge
}

// ProvideSignature appends a signature to a challenge so that it can be
// used to make requests to the MIG API.
func (ch Challenge) ProvideSignature(signature string) Token {
	return Token{
		token: ch.String() + ";" + signature,
	}
}

// String produces the string representation of a signed PGPAUTHORIZATION token.
func (tkn Token) String() string {
	return tkn.token
}

// GeneratePGPChallenge creates and records an unsigned token that the
// `PGPAuthorization` can receive an update for containing a signature.
func GeneratePGPChallenge() Challenge {
	max := big.NewInt(0x0FFFFFFFFFFFFFFD)
	nonce, err := rand.Int(rand.Reader, max)

	for err != nil {
		nonce, err = rand.Int(rand.Reader, nil)
	}

	currentTime := time.Now().UTC().String()

	return Challenge{
		challenge: fmt.Sprintf("%d;%s;%s", migAPIVersion, currentTime, nonce.String()),
	}
}

// StoreSignedToken records a token with a signature provided so that it can be
// used by `PGPAuthorization`
func (auth *PGPAuthorizer) StoreSignedToken(token Token) {
	auth.lock.Lock()
	defer auth.lock.Unlock()

	auth.token = token
}

// Authenticate modifies a request so that PGP-based authentication to the MIG API
// can take place.  If a token has not been set yet, an error will be returned.
func (auth PGPAuthorizer) Authenticate(req *http.Request) error {
	if auth.token == emptyToken() {
		return errors.New("PGPAuthorization cannot perform authorization before a signed token is set")
	}

	req.Header.Set(pgpAuthHeader, auth.token.String())
	return nil
}
