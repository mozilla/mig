// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package api

import (
	"github.com/gorilla/mux"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/api/actions"
)

// Dependencies contains all of the dependencies required to set up all of the
// request handlers for endpoints served by the API.
type Dependencies struct {
	ActionsCatalog *actions.Catalog
}

// MainRouterV1 constructs a router configured to serve the client daemon API.
func MainRouterV1(deps Dependencies) *mux.Router {
	createAction := actionsAPI.NewCreateHandler(deps.ActionsCatalog)
	readActionForSigning := actionsAPI.NewReadForSigningHandler(deps.ActionsCatalog)

	router := mux.NewRouter()
	router.Handle("/v1/actions/create", createAction).Methods("POST")
	router.Handle("/v1/actions/{id}/signing", readActionForSigning).Methods("GET")

	return router
}
