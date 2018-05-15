// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package dispatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"mig.ninja/mig"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
)

// APIDispatcher is a `Dispatcher` that will send actions to the MIG API.
type APIDispatcher struct {
	baseAddress string
}

// responseStructure is a convenient container type for the response
// format expected from the API.
type responseStructure struct {
	Collection collectionJSON `json:"collection"`
}

type collectionJSON struct {
	Error errorJSON `json:"error"`
}

type errorJSON struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewAPIDispatcher constructs a new `APIDispatcher`.
// The dispatcher needs to be informed of the base address of the MIG API,
// e.g. http://api.mig.ninja
func NewAPIDispatcher(serverURL string) APIDispatcher {
	return APIDispatcher{
		baseAddress: serverURL,
	}
}

// Dispatch sends a POST request to the MIG API to create an action.
func (dispatcher APIDispatcher) Dispatch(
	action mig.Action,
	auth authentication.Authenticator,
) error {
	// Construct the full path to the create action endpoint for v1 of the API.
	baseURL, parseErr := url.Parse(dispatcher.baseAddress)
	if parseErr != nil {
		return parseErr
	}
	reqPath, _ := url.Parse("/api/v1/action/create")
	fullURL := baseURL.ResolveReference(reqPath)

	// Create a reader for the JSON-encoded action.
	body, encodeErr := json.Marshal(action)
	if encodeErr != nil {
		return encodeErr
	}
	bodyReader := bytes.NewReader(body)

	// Create an authenticated request.
	request, createReqErr := http.NewRequest("POST", fullURL.String(), bodyReader)
	if createReqErr != nil {
		return createReqErr
	}
	request.Header.Set("Content-Type", "application/json")
	authErr := auth.Authenticate(request)
	if authErr != nil {
		return authErr
	}

	// Perform the request.
	client := &http.Client{}
	response, reqErr := client.Do(request)
	if reqErr != nil {
		return reqErr
	}

	// Check for an error in the response.
	if response.StatusCode != http.StatusAccepted {
		return errors.New("request not accepted by API")
	}
	respData := responseStructure{}
	decoder := json.NewDecoder(response.Body)
	defer response.Body.Close()
	decodeErr := decoder.Decode(&respData)
	if decodeErr != nil {
		return decodeErr
	}
	if respData.Collection.Error.Code != 0 {
		return errors.New(respData.Collection.Error.Message)
	}

	return nil
}
