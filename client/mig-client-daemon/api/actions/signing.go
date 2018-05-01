// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actionsAPI

import (
	"encoding/json"
	"net/http"

	//"github.com/gorilla/mux"

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
	Action string
}

// ReadForSigningHandler is an HTTP handler that handles requests to retrieve
// actions in a form in which a detached signature for them can be produced.
type ReadForSigningHandler struct {
	actionCatalog actions.Catalog
}

// NewReadForSigningHandler constructs a `ReadForSigningHandler`.
func NewReadForSigningHandler(catalog actions.Catalog) ReadForSigningHandler {
	return ReadForSigningHandler{
		actionCatalog: actions.Catalog,
	}
}

func (handler ReadForSigningHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	response := json.NewEncoder(res)

	response.Encode(&readForSigningResponse{
		Action: "",
	})
}
