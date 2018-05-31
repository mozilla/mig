// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"encoding/json"
	"fmt"
	"net/http"

	apiAuth "mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
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
	authenticator *apiAuth.PGPAuthorizer
}

// NewCompleteChallengeHandler constructs a `CompleteChallengeHandler`.
func NewCompleteChallengeHandler(auth *apiAuth.PGPAuthorizer) CompleteChallengeHandler {
	return CompleteChallengeHandler{
		authenticator: auth,
	}
}

func (handler CompleteChallengeHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	response := json.NewEncoder(res)
	request := completeChallengeRequest{}

	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := decoder.Decode(&request)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to decode request body. Error: %s", err.Error())
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&completeChallengeResponse{
			Error: &errMsg,
		})
		return
	}

	missingField := ""
	if request.Signature == "" {
		missingField = "signature"
	} else if request.Challenge == "" {
		missingField = "challenge"
	}
	if missingField != "" {
		errMsg := fmt.Sprintf("Missing JSON field in request body: \"%s\"", missingField)
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&completeChallengeResponse{
			Error: &errMsg,
		})
		return
	}

	challenge := apiAuth.NewChallenge(request.Challenge)
	token := challenge.ProvideSignature(request.Signature)
	handler.authenticator.StoreSignedToken(token)

	response.Encode(&completeChallengeResponse{
		Error: nil,
	})
}
