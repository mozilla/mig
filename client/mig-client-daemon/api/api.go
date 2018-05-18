// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package api

import (
	"github.com/gorilla/mux"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	actionsapi "mig.ninja/mig/client/mig-client-daemon/api/actions"
	authapi "mig.ninja/mig/client/mig-client-daemon/api/authentication"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
	"mig.ninja/mig/client/mig-client-daemon/migapi/dispatch"
)

// ActionDispatchDependencies contains the dependencies required to set up
// the action dispatch endpoint's request handler.
type ActionDispatchDependencies struct {
	Dispatcher    dispatch.ActionDispatcher
	Authenticator *authentication.PGPAuthorizer
}

// Dependencies contains all of the dependencies required to set up all of the
// request handlers for endpoints served by the API.
type Dependencies struct {
	ActionsCatalog *actions.Catalog
	ActionDispatch ActionDispatchDependencies
}

// RegisterRoutesV1 constructs and populates a subrouter based on `topRouter`
// with a path prefix of "/v1".
func RegisterRoutesV1(topRouter *mux.Router, deps Dependencies) {
	getPGPChallenge := authapi.NewGetChallengeHandler()
	completePGPChallenge := authapi.NewCompleteChallengeHandler(deps.ActionDispatch.Authenticator)
	createAction := actionsapi.NewCreateHandler(deps.ActionsCatalog)
	retrieveAction := actionsapi.NewRetrieveHandler(deps.ActionsCatalog)
	readActionForSigning := actionsapi.NewReadForSigningHandler(deps.ActionsCatalog)
	signAction := actionsapi.NewProvideSignatureHandler(deps.ActionsCatalog)
	dispatchAction := actionsapi.NewDispatchHandler(deps.ActionsCatalog, deps.ActionDispatch.Dispatcher, deps.ActionDispatch.Authenticator)

	router := topRouter.PathPrefix("/v1").Subrouter()
	router.Handle("/authentication/pgp", getPGPChallenge).Methods("GET")
	router.Handle("/authentication/pgp", completePGPChallenge).Methods("POST")
	router.Handle("/actions/create", createAction).Methods("POST")
	router.Handle("/actions/{id}", retrieveAction).Methods("GET")
	router.Handle("/actions/{id}/signing", readActionForSigning).Methods("GET")
	router.Handle("/actions/{id}/sign", signAction).Methods("PUT")
	router.Handle("/actions/{id}/dispatch", dispatchAction).Methods("PUT")
}
