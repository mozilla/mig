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

// RegisterRoutesV1 constructs and populates a subrouter based on `topRouter`
// with a path prefix of "/v1".
func RegisterRoutesV1(topRouter *mux.Router, deps Dependencies) {
	createAction := actionsAPI.NewCreateHandler(deps.ActionsCatalog)
	readActionForSigning := actionsAPI.NewReadForSigningHandler(deps.ActionsCatalog)

	router := topRouter.PathPrefix("/v1").Subrouter()
	router.Handle("/actions/create", createAction).Methods("POST")
	router.Handle("/actions/{id}/signing", readActionForSigning).Methods("GET")
}
