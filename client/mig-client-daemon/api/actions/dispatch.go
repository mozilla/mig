// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"encoding/json"
	"net/http"

	//"github.com/gorilla/mux"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
	"mig.ninja/mig/client/mig-client-daemon/migapi/dispatch"
)

// dispatchRequest contains the parameters of a request to dispatch an action.
type dispatchRequest struct {
	ActionID ident.Identifier
}

// dispatchResponse contains the body of a response to a dispatch request.
type dispatchResponse struct {
	Error  *string `json:"error"`
	Status string  `json:"status"`
}

// DispatchHandler is an HTTP handler for requests to have an action dispatched
// to the MIG API.
type DispatchHandler struct {
	actionCatalog *actions.Catalog
	dispatcher    dispatch.ActionDispatcher
	authenticator authentication.Authenticator
}

// NewDispatchHandler cponstructs a new `DispatchHandler`.
func NewDispatchHandler(
	catalog *actions.Catalog,
	dispatcher dispatch.ActionDispatcher,
	authenticator authentication.Authenticator,
) DispatchHandler {
	return DispatchHandler{
		actionCatalog: catalog,
		dispatcher:    dispatcher,
		authenticator: authenticator,
	}
}

func (handler DispatchHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	response := json.NewEncoder(res)

	//urlVars := mux.Vars(req)
	//request := dispatchRequest{
	//	ActionID: ident.Identifier(urlVars["id"]),
	//}

	response.Encode(&dispatchResponse{
		Error:  nil,
		Status: "dispatched",
	})
}
