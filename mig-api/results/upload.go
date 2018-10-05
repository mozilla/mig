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

// result is used to decouple our request data type from the rest of MIG's types.
// The toModuleResult method provides a convenient conversion to modules.Result.
type result struct {
	FoundAnything bool        `json:"foundAnything"`
	Success       bool        `json:"success"`
	Elements      interface{} `json:"elements"`
	Statistics    interface{} `json:"statistics"`
	Errors        []string    `json:"errors"`
}

// uploadRequest contains the body of a request to the Upload handler.
type uploadRequest struct {
	Action  float64  `json:"action"`
	Results []result `json:"results"`
}

// uploadResponse contains the body of a response to a request to upload results.
type uploadResponse struct {
	Error *string `json:"error"`
}

// NewUpload constructs a new Upload.
func NewUpload(persist PersistResult) Upload {
	return Upload{
		persist: persist,
	}
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

// toModuleResult converts instances of our result abstraction into a Result
// used by the rest of MIG.
func (res result) toModuleResult() modules.Result {
	return modules.Result{
		FoundAnything: res.FoundAnything,
		Success:       res.Success,
		Elements:      res.Elements,
		Statistics:    res.Statistics,
		Errors:        res.Errors,
	}
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

	results := make([]module.Result, len(reqData.Results))
	for i, res := range reqData.Results {
		results[i] = reqData.Results[i].toModuleResult()
	}

	persistErr := handler.persist.PersistResult(reqData.Action, results)
	if persistErr != nil {
		errMsg := fmt.Sprintf("Failed to save results: %s", persistErr.Error())
		response.WriteHeader(http.StatusInternalServerError)
		resEncoder.Encode(&uploadResponse{&errMsg})
		return
	}

	resEncoder.Encode(&uploadResponse{nil})
}
