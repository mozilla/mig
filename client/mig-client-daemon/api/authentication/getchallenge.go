// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"encoding/json"
	"net/http"
	
	apiAuth "mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
)

// getChallengeRequest stores parameters for a request to receive a PGP auth
// challenge.
type getChallengeRequest struct {}

// getChallengeResponse stores the body of a response to a request to retrieve
// a PGP auth challenge.
type getChallengeResponse struct {
	Challenge string `json:"challenge"`
}

// GetChallengeHandler is an HTTP handler that serves requests to retrieve a
// PGP auth challenge.
type GetChallengeHandler struct {}

// NewGetChallengeHandler constructs a `GetChallengeHandler`.
func NewGetChallengeHandler() GetChallengeHandler {
	return GetChallengeHandler{}
}

func (handler GetChallengeHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	response := json.NewEncoder(res)

	token := apiAuth.GeneratePGPChallenge()

	response.Encode(&getChallengeResponse{
		Challenge: token.String(),
	})
}