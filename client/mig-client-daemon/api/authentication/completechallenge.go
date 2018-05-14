// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"encoding/json"
	"net/http"
)

// completeChallengeRequest contains the body of a request to complete a
// challenge for PGP-based authentication to the MIG API.
type completeChallengeRequest struct {
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

// completeChallengeResponse contains the body of a response to a request
// to complete a PGP-based authentication challenge.
type completeChallengeResponse struct {
	Error *string `json:"error"`
}

// CompleteChallengeHandler is an HTTP handler that serves requests to
// complete challenges for PGP-based authentication to the MIG API.
type CompleteChallengeHandler struct {
}

// NewCompleteChallengeHandler constructs a `CompleteChallengeHandler`.
func NewCompleteChallengeHandler() CompleteChallengeHandler {
	return CompleteChallengeHandler{}
}

func (handler CompleteChallengeHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	response := json.NewEncoder(res)

	response.Encode(&completeChallengeResponse{
		Error: nil,
	})
}
