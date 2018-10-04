// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package results

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mozilla/mig/modules"
)

// PersistResults abstracts over operations that allow the MIG API to
// save results produced while executing an action.
type PersistResults interface {
	PersistResults(float64, []modules.Result) error
}

// Upload is an HTTP request handler that serves PUT requests
// containing results produced while executing an action.
type Upload struct {
	persist PersistResult
}

// NewUpload constructs a new Upload.
func NewUpload(persist PersistResult) Upload {
	return Upload{
		persist: persist,
	}
}

// uploadRequest contains the body of a request to the Upload handler.
type uploadRequest struct {
	Action  float64          `json:"action"`
	Results []modules.Result `json:"results"`
}

// uploadResponse contains the body of a response to a request to upload results.
type uploadResponse struct {
	Error *string `json:"error"`
}

// validate ensures that a results upload request contains all of the data
// required to satisfy the request.
func (req uploadRequest) validate() error {
	if req.Action == "" {
		return fmt.Errorf("missing action field")
	}

	if len(req.Results) == 0 {
		return fmt.Errorf("empty array of results")
	}

	return nil
}

func (handler Upload) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	reqData := uploadRequest{}
	decoder := json.NewDecoder(request.Body)
	resEncoder := json.NewEncoder(response)

	response.Header().Set("Content-Type", "application/json")

	defer request.Body.Close()

	decodeErr := decoder.Decode(&reqData)
	if decodeErr != nil {
		errMsg := fmt.Sprintf("Failed to decode request body: %s", decodeErr.Error())
		response.WriteHeader(http.StatusBadRequest)
		resEncoder.Encode(&uploadResponse{&errMsg})
		return
	}

	validateErr := reqData.validate()
	if validateErr != nil {
		errMsg := fmt.Sprintf("Missing or invalid data in request: %s", validateErr.Error())
		response.WriteHeader(http.StatusBadRequest)
		resEncoder.Encode(&uploadResponse{&errMsg})
		return
	}

	persistErr := handler.persist.PersistResult(reqData.Action, reqData.Results)
	if persistErr != nil {
		errMsg := fmt.Sprintf("Failed to save results: %s", persistErr.Error())
		response.WriteHeader(http.StatusInternalServerError)
		resEncoder.Encode(&uploadResponse{&errMsg})
		return
	}

	resEncoder.Encode(&uploadResponse{nil})
}
