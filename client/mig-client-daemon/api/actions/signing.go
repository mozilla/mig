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

// readForSigningRequest contains the positional parameter of a request to
// retrieve an action so that a signature for it can be produced.
type readForSigningRequest struct {
	Action ident.Identifier
}

// readForSigningResponse contains the body of a response to a request to
// retrieve an action in a signable form.
type readForSigningResponse struct {
	Error  *string `json:"error"`
	Action string  `json:"action"`
}

// ReadForSigningHandler is an HTTP handler that handles requests to retrieve
// actions in a form in which a detached signature for them can be produced.
type ReadForSigningHandler struct {
	actionCatalog *actions.Catalog
}

// NewReadForSigningHandler constructs a `ReadForSigningHandler`.
func NewReadForSigningHandler(catalog *actions.Catalog) ReadForSigningHandler {
	return ReadForSigningHandler{
		actionCatalog: catalog,
	}
}

func (handler ReadForSigningHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	response := json.NewEncoder(res)
	urlVars := mux.Vars(req)

	reqData := readForSigningRequest{
		Action: ident.Identifier(urlVars["id"]),
	}

	action, found := handler.actionCatalog.Lookup(reqData.Action)
	if !found {
		errMsg := fmt.Sprintf("No such action %s.", string(reqData.Action))
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&readForSigningResponse{
			Error:  &errMsg,
			Action: "",
		})
		return
	}

	actionStr, err := action.String()
	if err != nil {
		errMsg := err.Error()
		response.Encode(&readForSigningResponse{
			Error:  &errMsg,
			Action: "",
		})
		return
	}

	response.Encode(&readForSigningResponse{
		Error:  nil,
		Action: actionStr,
	})
}
