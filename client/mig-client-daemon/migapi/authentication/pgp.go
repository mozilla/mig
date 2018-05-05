// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"net/http"
	"time"
)

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
	unsignedToken    *UnsignedToken
	signedToken      *Token
	tokenGeneratedAt *time.Time
}

// GenerateUnsignedToken creates and records an unsigned token that the
// `PGPAuthorization` can receive an update for containing a signature.
func (auth *PGPAuthorization) GenerateUnsignedToken() UnsignedToken {
	return UnsignedToken{
		token: "",
	}
}

// StoreSignedToken records a token with a signature provided so that it can be
// used by `PGPAuthorization`
func (auth *PGPAuthorization) StoreSignedToken(token Token) error {
	return nil
}

// ProvideSignature appends a signature to an unsigned token so that it can be
// used to make requests to the MIG API.
func (tkn UnsignedToken) ProvideSignature(signature string) Token {
	return Token{
		token: tkn.token + ";" + signature,
	}
}

func (auth PGPAuthorization) Authenticate(req *http.Request) error {
	return nil
}
