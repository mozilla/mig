// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actionsAPI

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

// createRequest contains the body of a request to create an action.
type createRequest struct {
	ModuleName          string                   `json:"module"`
	ExpireAfterSeconds  uint64                   `json:"expireAfter"`
	TargetQueries       []map[string]interface{} `json:"target"`
	ModuleConfiguration map[string]interface{}   `json:"moduleConfig"`
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
	actionCatalog actions.Catalog
}

// NewCreateHandler constructs a `CreateHandler`.
func NewCreateHandler(catalog actions.Catalog) CreateHandler {
	return CreateHandler{
		actionCatalog: catalog,
	}
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

	queryIndex := 0
	queries := make([]targeting.Query, len(request.TargetQueries))
	for _, queryConfig := range request.TargetQueries {
		query, err := targeting.FromMap(queryConfig)
		if err != nil {
			errMsg := fmt.Sprintf("Invalid agent target query data found. Error: %s", err.Error())
			res.WriteHeader(http.StatusBadRequest)
			response.Encode(&createResponse{
				Error:  &errMsg,
				Action: ident.EmptyID,
			})
			return
		}

		queries[queryIndex] = query
		queryIndex++
	}

	module, err := modules.FromMap(request.ModuleName, request.ModuleConfiguration)
	if err != nil {
		errMsg := fmt.Sprintf("Invalid module configuration. Error: %s", err.Error())
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&createResponse{
			Error:  &errMsg,
			Action: ident.EmptyID,
		})
		return
	}

	expireAfter := time.Duration(request.ExpireAfterSeconds) * time.Second
	actionID, err := handler.actionCatalog.Create(module, queries, expireAfter)
	if err != nil {
		errMsg := fmt.Sprintf("Could not create action. Error: %s", err.Error())
		res.WriteHeader(http.StatusBadRequest)
		response.Encode(&createResponse{
			Error:  &errMsg,
			Action: ident.EmptyID,
		})
		return
	}

	response.Encode(&createResponse{
		Error:  nil,
		Action: actionID,
	})
}
