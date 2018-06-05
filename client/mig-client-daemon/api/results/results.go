// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package results

import (
	"encoding/json"
	"net/http"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/modules"
)

type searchResultsRequest struct {
	ActionID actions.InternalActionID
}

type searchResultsResponse struct {
	Error   *string          `json:"error"`
	Results []modules.Result `json:"results"`
}

type SearchResultsHandler struct {
}

func NewSearchResultsHandler() SearchResultsHandler {
	return SearchResultsHandler{}
}

func (handler SearchResultsHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	response := json.NewEncoder(res)

	actionIDs := req.URL.Query()["action"]
	if len(actionIDs) == 0 {
		res.WriteHeader(http.StatusBadRequest)
		errMsg := "missing action parameter"
		response.Encode(&searchResultsResponse{
			Error:   &errMsg,
			Results: []modules.Result{},
		})
		return
	}

	response.Encode(&searchResultsResponse{
		Error:   nil,
		Results: []modules.Result{},
	})
}
