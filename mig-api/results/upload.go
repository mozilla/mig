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

	"github.com/mozilla/mig/modules"
)

// PersistError represents each of the possible failures that may occur
// within an implementation of the PersistResults interface.
// Think of this as an enum, with each const declared with the prefix
// `PersistError` being a variant of this enum.
// This type implements the error interface.
type PersistError uint

const (
	// PersistErrorNil indicates that no error occurred.
	PersistErrorNil = iota

	// PersistErrorInvalidAction indicates that the action specified does not exist.
	PersistErrorInvalidAction PersistError = iota

	// PersistErrorNotAuthorized indicates that the agent was not authorized to save results.
	PersistErrorNotAuthorized PersistError = iota

	// PersistErrorMediumFailure indicates that the results could not be saved due to
	// an error relating to the medium they would be saved to (e.g. disk, database).
	PersistErrorMediumFailure PersistError = iota
)

// PersistResults abstracts over operations that allow the MIG API to
// save results produced while executing an action.
type PersistResults interface {
	PersistResults(float64, []modules.Result) PersistError
}

// Upload is an HTTP request handler that serves PUT requests
// containing results produced while executing an action.
type Upload struct {
	persist PersistResults
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
func NewUpload(persist PersistResults) Upload {
	return Upload{
		persist: persist,
	}
}

// Error implements the error interface for PersistError.
func (err PersistError) Error() string {
	switch err {
	case PersistErrorNil:
		return ""
	case PersistErrorInvalidAction:
		return "invalid action does not exist"
	case PersistErrorNotAuthorized:
		return "not authorized"
	case PersistErrorMediumFailure:
		return "failed to write to medium"
	default:
		return ""
	}
}

// validate ensures that a results upload request contains all of the data
// required to satisfy the request.
func (req uploadRequest) validate() error {
	if req.Action == 0.0 {
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

	results := make([]modules.Result, len(reqData.Results))
	for i, res := range reqData.Results {
		results[i] = res.toModuleResult()
	}

	persistErr := handler.persist.PersistResults(reqData.Action, results)
	if persistErr != PersistErrorNil {
		errMsg := fmt.Sprintf("Failed to save results: %s", persistErr.Error())

		switch persistErr {
		case PersistErrorMediumFailure:
			response.WriteHeader(http.StatusInternalServerError)
		case PersistErrorNotAuthorized:
			response.WriteHeader(http.StatusUnauthorized)
		default:
			response.WriteHeader(http.StatusBadRequest)
		}

		resEncoder.Encode(&uploadResponse{&errMsg})
		return
	}

	resEncoder.Encode(&uploadResponse{nil})
}
