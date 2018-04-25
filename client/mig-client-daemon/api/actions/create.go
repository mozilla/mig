// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"encoding/json"
	"fmt"
	"net/http"

	"mig.ninja/mig/client/mig-client-daemon/ident"
)

// createRequest contains the body of a request to create an action.
type createRequest struct {
	ModuleName          string                 `json:"module"`
	ExpireAfterSeconds  uint64                 `json:"expireAfter"`
	AgentsToTarget      string                 `json:"target"`
	ModuleConfiguration map[string]interface{} `json:"moduleConfig"`
}

// createResponse contains the body of a response to a request to have an
// action created.
type createResponse struct {
	Error  *string          `json:"error"`
	Action ident.Identifier `json:"action"`
}

// CreateHandler is an HTTP handler that will attempt to handle requests to
// have an action created by an investigator.
type CreateHandler struct {
}

// NewCreateHandler constructs a `CreateHandler`.
func NewCreateHandler() CreateHandler {
	return CreateHandler{}
}

func (handler CreateHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	request := createRequest{}
	decoder := json.NewDecoder(req.Body)
	response := json.NewEncoder(res)
	defer req.Body.Close()

	decodeError := decoder.Decode(&request)
	if decodeError != nil {
		errMsg := fmt.Sprintf("Failed to decode request body. Error: %s", decodeError.Error())
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&createResponse{
			Error:  &errMsg,
			Action: ident.EmptyID,
		})
		return
	}

	madeUpID := ident.Identifier("Testing. Be sure to fix me later!")
	response.Encode(&createResponse{
		Error:  nil,
		Action: madeUpID,
	})
}
