// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package migapi

import (
	"net/http"
)

// UnsignedToken is the base of a token that a signature can be provided for
// and then uploaded into an `Authenticator`.
type UnsignedToken struct {
	token string
}

// Token contains a signed token string passed to the MIG API to prove that
// the sender is an investigator known by the API.
type Token {
	token string
}

// Autnenticator provides services for managing an authentication token
// required to interact with the MIG API.
type Authenticator interface {
	// IsAuthenticated determines whether the token being managed by the
	// `Authenticator` requires replacing.
	IsAuthenticated() bool

	// GenerateToken creates a base token that a signature can be provided for.
	GenerateToken() UnsignedToken
	
	// Authenticate should replace the signed token being managed.
	Authenticate(*http.Request, Token)
}

// ActionDispatcher provides a service for dispatching actions to the MIG API.
type ActionDispatcher interface {
	// Dispatch sends an action to the MIG API.
	Dispatch(mig.Action, Authenticator) error
}

// ProvideSignature appends a signature to an unsigned token so that it can be
// used to make requests to the MIG API.
func (tkn UnsignedToken) ProvideSignature(signature string) Token {
	return Token {
		tkn.token + ";" + signature
	}
}
