// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actionsAPI

import (
	"encoding/json"
	"net/http"

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
	response := json.NewEncoder(res)

	response.Encode(&provideSignatureResponse{
		Error: nil,
	})
}
