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

	"github.com/gorilla/mux"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
)

// provideSignatureRequest contains the contents of a request to have a
// signature added to an action.
type provideSignatureRequest struct {
	Signature string           `json:"signature"`
	ActionID  ident.Identifier // Not loaded from JSON
}

// provideSignatureResponse contains the body of a response indicating whether
// the provided signature was added to the action specified.
type provideSignatureResponse struct {
	Error *string `json:"error"`
}

// ProvideSignatureHandler is an HTTP handler that handles requests to upload a
// signature for an action.
type ProvideSignatureHandler struct {
	actionCatalog *actions.Catalog
}

// NewProvideSignatureHandler constructs a `ProvideSignatureHandler`.
func NewProvideSignatureHandler(catalog *actions.Catalog) ProvideSignatureHandler {
	return ProvideSignatureHandler{
		actionCatalog: catalog,
	}
}

func (handler ProvideSignatureHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	request := provideSignatureRequest{}
	response := json.NewEncoder(res)

	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	decodeErr := decoder.Decode(&request)
	if decodeErr != nil {
		errMsg := fmt.Sprintf("Failed to decode request body. Error: %s", decodeErr.Error())
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&provideSignatureResponse{
			Error: &errMsg,
		})
		return
	}

	if request.Signature == "" {
		errMsg := "Empty or missing signature."
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&provideSignatureResponse{
			Error: &errMsg,
		})
		return
	}

	urlVars := mux.Vars(req)
	request.ActionID = ident.Identifier(urlVars["id"])

	err := handler.actionCatalog.AddSignature(request.ActionID, request.Signature)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to add signature. Error: %s", err.Error())
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&provideSignatureResponse{
			Error: &errMsg,
		})
		return
	}

	response.Encode(&provideSignatureResponse{
		Error: nil,
	})
}
