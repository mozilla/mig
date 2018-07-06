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

	"mig.ninja/mig"
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
)

// retrieveRequest contains the contents of a request to retrieve an action.
type retrieveRequest struct {
	ActionID ident.Identifier
}

// retrieveResponse contains an action if found or else an error if not.
type retrieveResponse struct {
	Error  *string     `json:"error"`
	Action *mig.Action `json:"action"`
}

// RetrieveHandler is a request handler that serves actions encoded as JSON.
type RetrieveHandler struct {
	actionsCatalog *actions.Catalog
}

// NewRetrieveHandler constructs a new `RetrieveHandler`.
func NewRetrieveHandler(catalog *actions.Catalog) RetrieveHandler {
	return RetrieveHandler{
		actionsCatalog: catalog,
	}
}

func (handler RetrieveHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	response := json.NewEncoder(res)

	urlValues := mux.Vars(req)
	request := retrieveRequest{
		ActionID: ident.Identifier(urlValues["id"]),
	}

	record, found := handler.actionsCatalog.Lookup(request.ActionID)
	if !found {
		errMsg := fmt.Sprintf("No such action %s", string(request.ActionID))
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&retrieveResponse{
			Error:  &errMsg,
			Action: nil,
		})
		return
	}

	response.Encode(&retrieveResponse{
		Error:  nil,
		Action: &record.Action,
	})
}
